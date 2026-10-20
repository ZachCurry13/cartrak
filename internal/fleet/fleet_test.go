package fleet

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/config"
	"github.com/ZachCurry13/cartrak/internal/store"
	"golang.org/x/crypto/bcrypt"
)

// start makes a data folder that looks like an installation from before several cars (one database with two people and
// a trip), starts the fleet on it with the demo provider, and returns a test server in front of it.
func start(t *testing.T) (*Fleet, *httptest.Server, string) {
	t.Helper()
	dir := t.TempDir()
	old, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	h, _ := bcrypt.GenerateFromPassword([]byte("correct-horse-battery"), bcrypt.MinCost)
	_ = old.CreateUser("ann", string(h), "admin")
	_ = old.CreateUser("bob", string(h), "viewer")
	_ = old.CreateUser("cat", string(h), "viewer")
	_, _ = old.DB.Exec(`INSERT INTO trips(start_ts,end_ts,distance_km,open) VALUES(100,200,12.5,0)`)
	_ = old.Close()

	cfg := &config.Config{Addr: ":0", DataDir: dir, Provider: "demo", TZ: time.UTC,
		PollDriving: time.Hour, PollCharge: time.Hour, PollParked: time.Hour, PollLive: time.Hour}
	f, err := New(cfg, "test")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	if err := f.Run(ctx); err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(f.Handler())
	t.Cleanup(func() { ts.Close(); cancel(); f.Close() })
	return f, ts, dir
}

type client struct {
	t    *testing.T
	base string
	c    *http.Client
}

func login(t *testing.T, base, name string) *client {
	jar, _ := cookiejar.New(nil)
	c := &client{t: t, base: base, c: &http.Client{Jar: jar}}
	if code, body := c.do("POST", "/api/login", `{"name":"`+name+`","password":"correct-horse-battery"}`, nil); code != 200 {
		t.Fatalf("login %s: %d %s", name, code, body)
	}
	return c
}

func (c *client) do(method, path, body string, hdr map[string]string) (int, string) {
	req, _ := http.NewRequest(method, c.base+path, strings.NewReader(body))
	req.Header.Set("X-Requested-With", "cartrak")
	req.Header.Set("Content-Type", "application/json")
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	resp, err := c.c.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func car(id string) map[string]string { return map[string]string{"X-Car": id} }

func TestMigrationKeepsPeopleAndMakesTheFirstCar(t *testing.T) {
	f, _, dir := start(t)
	cars, _ := f.Core.ListCars()
	if len(cars) != 1 || cars[0].ID != 1 || cars[0].Dir != "" {
		t.Fatalf("cars: %+v", cars)
	}
	ann, _ := f.Core.UserByName("ann")
	bob, _ := f.Core.UserByName("bob")
	if f.Core.Access(ann.ID, 1) != store.AccessOwner || f.Core.Access(bob.ID, 1) != store.AccessViewer {
		t.Fatal("people should keep their place: administrators own the first car, others read it")
	}
	backups, _ := filepath.Glob(filepath.Join(dir, "backups", "before-several-cars-*.db"))
	if len(backups) != 1 {
		t.Fatalf("a backup of the old database should be made first, found %v", backups)
	}
	if _, err := os.Stat(filepath.Join(dir, store.CoreFile)); err != nil {
		t.Fatal("core database missing")
	}
}

func TestEachCarSeesOnlyItsOwnDataAndPeopleSeeOnlyTheirCars(t *testing.T) {
	f, ts, dir := start(t)
	ann := login(t, ts.URL, "ann")
	bob := login(t, ts.URL, "bob")
	cat := login(t, ts.URL, "cat")

	// the first car has the old trip; ann adds a second car
	if code, body := ann.do("GET", "/api/trips", "", car("1")); code != 200 || !strings.Contains(body, "12.5") {
		t.Fatalf("car 1 trips: %d %s", code, body)
	}
	code, body := ann.do("POST", "/api/cars", `{"name":"Truck"}`, nil)
	if code != 200 {
		t.Fatalf("add car: %d %s", code, body)
	}
	if _, err := os.Stat(filepath.Join(dir, "cars", "2", "cartrak.db")); err != nil {
		t.Fatal("the second car needs a database of its own")
	}
	if code, body := ann.do("GET", "/api/trips", "", car("2")); code != 200 || strings.Contains(body, "12.5") {
		t.Fatalf("car 2 must not see car 1's trip: %d %s", code, body)
	}

	// bob reads car 1 only; he may not add a car, nor open car 2
	var list struct {
		Cars   []struct{ ID int64 } `json:"cars"`
		CanAdd bool                 `json:"canAdd"`
	}
	_, body = bob.do("GET", "/api/cars", "", nil)
	_ = json.Unmarshal([]byte(body), &list)
	if len(list.Cars) != 1 || list.Cars[0].ID != 1 || list.CanAdd {
		t.Fatalf("bob's cars: %s", body)
	}
	if code, _ := bob.do("GET", "/api/trips", "", car("2")); code != 403 {
		t.Fatalf("bob on car 2 = %d, want 403", code)
	}
	if code, _ := bob.do("POST", "/api/cars", `{"name":"x"}`, nil); code != 403 {
		t.Fatalf("bob adding a car = %d, want 403", code)
	}
	if code, _ := bob.do("GET", "/api/trips", "", nil); code != 200 {
		t.Fatalf("bob's default car = %d", code)
	}

	// a viewer cannot do admin things even on a car he can read
	if code, _ := bob.do("POST", "/api/poll", "", car("1")); code != 403 {
		t.Fatalf("viewer poll = %d, want 403", code)
	}

	// cat is allowed to bring a car: she adds one, owns it, can manage it, but cannot touch car 1 or the people
	catID := func() int64 { u, _ := f.Core.UserByName("cat"); return u.ID }()
	_ = f.Core.SetOwnCar(catID, true)
	cat = login(t, ts.URL, "cat") // a fresh sign-in picks up the flag
	if code, body := cat.do("POST", "/api/cars", `{"name":"Cat's car"}`, nil); code != 200 {
		t.Fatalf("cat add car: %d %s", code, body)
	}
	if code, _ := cat.do("POST", "/api/poll", "", car("3")); code != 200 {
		t.Fatalf("an owner manages her own car: %d", code)
	}
	if code, _ := cat.do("POST", "/api/poll", "", car("1")); code != 403 {
		t.Fatalf("cat on car 1 = %d, want 403", code)
	}
	if code, _ := cat.do("PUT", "/api/admin/settings", `{"values":{"CARTRAK_TUNNEL_TOKEN":"stolen"}}`, car("3")); code != 403 {
		t.Fatalf("an owner must not change remote access: %d", code)
	}
	if code, _ := cat.do("POST", "/api/restore/apply", `{"confirm":"RESTORE"}`, car("3")); code != 403 {
		t.Fatalf("an owner must not stop CarTrak for everyone: %d", code)
	}
	if code, _ := cat.do("GET", "/api/users", "", nil); code != 403 {
		t.Fatalf("an owner must not manage people: %d", code)
	}
	if code, _ := cat.do("GET", "/api/debug-report", "", car("3")); code != 200 {
		t.Fatalf("an owner can read her car's debug report: %d", code)
	}
	if code, _ := cat.do("GET", "/api/debug-report", "", car("2")); code != 403 {
		t.Fatalf("cat on car 2 = %d, want 403", code)
	}

	// the administrator can open any car, and its owner is told
	if code, _ := ann.do("GET", "/api/trips", "", car("3")); code != 200 {
		t.Fatalf("an administrator can open any car: %d", code)
	}
	if _, body := cat.do("GET", "/api/cars", "", nil); !strings.Contains(body, `"adminViews":{"ann"`) {
		t.Fatalf("the owner should be told an administrator looked: %s", body)
	}
	if _, body := ann.do("GET", "/api/cars", "", nil); strings.Contains(body, "adminViews") {
		t.Fatalf("only an owner is told: %s", body)
	}
}

func TestOwnerSharesACarAsReadOnlyAndCannotMakeOwners(t *testing.T) {
	f, ts, _ := start(t)
	ann := login(t, ts.URL, "ann")
	bob := login(t, ts.URL, "bob")
	catU, _ := f.Core.UserByName("cat")
	bobU, _ := f.Core.UserByName("bob")
	_ = f.Core.SetOwnCar(catU.ID, true)
	cat := login(t, ts.URL, "cat")
	if code, _ := cat.do("POST", "/api/cars", `{"name":"Cat's car"}`, nil); code != 200 {
		t.Fatal("add car")
	}
	// cat shares it with bob (read only)
	if code, body := cat.do("PUT", "/api/cars/2/access", `{"userId":`+itoa(bobU.ID)+`,"level":"viewer"}`, nil); code != 200 {
		t.Fatalf("share: %d %s", code, body)
	}
	if code, _ := bob.do("GET", "/api/trips", "", car("2")); code != 200 {
		t.Fatal("bob should read the shared car")
	}
	if code, _ := bob.do("POST", "/api/poll", "", car("2")); code != 403 {
		t.Fatal("a viewer cannot poll")
	}
	// cat cannot make bob an owner; ann can
	if code, _ := cat.do("PUT", "/api/cars/2/access", `{"userId":`+itoa(bobU.ID)+`,"level":"owner"}`, nil); code != 403 {
		t.Fatalf("an owner must not make owners: %d", code)
	}
	if code, _ := ann.do("PUT", "/api/cars/2/access", `{"userId":`+itoa(bobU.ID)+`,"level":"owner"}`, nil); code != 200 {
		t.Fatal("an administrator can make owners")
	}
	// alerts: bob can now be told about car 2 but cat's other friend cannot
	if !f.canSee(bobU.ID, 2) || f.canSee(999, 2) {
		t.Fatal("notification visibility follows access")
	}
}

func TestCarCannotBeMadeBeyondTheLimitAndTheFirstCannotBeRemoved(t *testing.T) {
	_, ts, _ := start(t)
	ann := login(t, ts.URL, "ann")
	for i := 2; i <= MaxCars; i++ {
		if code, body := ann.do("POST", "/api/cars", `{"name":"c"}`, nil); code != 200 {
			t.Fatalf("car %d: %d %s", i, code, body)
		}
	}
	if code, _ := ann.do("POST", "/api/cars", `{"name":"one too many"}`, nil); code != 409 {
		t.Fatalf("beyond the limit = %d, want 409", code)
	}
	if code, _ := ann.do("DELETE", "/api/cars/1", "", nil); code != 409 {
		t.Fatalf("removing the first car = %d, want 409", code)
	}
	if code, _ := ann.do("DELETE", "/api/cars/2", "", nil); code != 200 {
		t.Fatal("removing another car")
	}
	if code, _ := ann.do("GET", "/api/trips", "", car("2")); code == 200 {
		t.Fatal("a removed car must not answer")
	}
}

func itoa(n int64) string { b, _ := json.Marshal(n); return string(b) }

func TestMigrationClearsTheOldAccountsAndBackupsCarryNone(t *testing.T) {
	f, ts, _ := start(t)
	if f.first.CountUsers() != 0 {
		t.Fatal("the old copy of the accounts should be cleared once copied")
	}
	if f.Core.CountUsers() != 3 {
		t.Fatalf("the core should hold the accounts: %d", f.Core.CountUsers())
	}
	ann := login(t, ts.URL, "ann")
	if code, _ := ann.do("POST", "/api/cars", `{"name":"Truck"}`, nil); code != 200 {
		t.Fatal("add car")
	}
	// a car's own backup has no accounts
	req, _ := http.NewRequest("GET", ts.URL+"/api/backup", nil)
	req.Header.Set("X-Car", "2")
	resp, err := ann.c.Do(req)
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("backup: %v %v", err, resp)
	}
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	p := filepath.Join(t.TempDir(), "b.db")
	_ = os.WriteFile(p, b, 0o644)
	info, err := store.InspectBackup(p)
	if err != nil || len(info.Users) != 0 {
		t.Fatalf("a car's backup must open and carry no accounts: %v %+v", err, info)
	}
	// the whole-installation zip has the core and every car
	resp, err = ann.c.Get(ts.URL + "/api/backup/site")
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("site backup: %v %v", err, resp)
	}
	zb, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	zr, err := zip.NewReader(bytes.NewReader(zb), int64(len(zb)))
	if err != nil {
		t.Fatal(err)
	}
	have := map[string]bool{}
	for _, zf := range zr.File {
		have[zf.Name] = true
	}
	for _, want := range []string{"core.db", "car-1.db", "car-2.db", "cars.json"} {
		if !have[want] {
			t.Fatalf("the zip is missing %s: %v", want, have)
		}
	}
	// a person who is not a site administrator cannot take it
	bob := login(t, ts.URL, "bob")
	if code, _ := bob.do("GET", "/api/backup/site", "", nil); code != 403 {
		t.Fatalf("site backup for a viewer = %d, want 403", code)
	}
}

func TestEachCarsRestoreFolderIsItsOwn(t *testing.T) {
	f, ts, dir := start(t)
	ann := login(t, ts.URL, "ann")
	if code, _ := ann.do("POST", "/api/cars", `{"name":"Truck"}`, nil); code != 200 {
		t.Fatal("add car")
	}
	// put a backup of car 1 in car 2's restore folder and start the fleet again: it must be applied to car 2 only
	cdir := filepath.Join(dir, "cars", "2")
	if err := os.MkdirAll(filepath.Join(cdir, "restore"), 0o755); err != nil {
		t.Fatal(err)
	}
	snap := filepath.Join(t.TempDir(), "snap.db")
	if err := f.Runtime(1).S.BackupCar(snap, false); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(snap)
	_ = os.WriteFile(filepath.Join(cdir, "restore", store.PendingName), data, 0o644)
	_ = os.WriteFile(filepath.Join(cdir, "restore", store.ApplyFlag), []byte("1"), 0o644)
	ts.Close()
	f.Close()

	cfg := f.Cfg
	f2, err := New(cfg, "test")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer func() { cancel(); f2.Close() }()
	if err := f2.Run(ctx); err != nil {
		t.Fatal(err)
	}
	t1, _ := f2.Runtime(1).S.ListTrips(store.TripFilter{Limit: 5})
	t2, _ := f2.Runtime(2).S.ListTrips(store.TripFilter{Limit: 5})
	if len(t1) != 1 || len(t2) != 1 {
		t.Fatalf("car 2 should now hold the restored trip and car 1 keep its own: %d %d", len(t1), len(t2))
	}
}

func TestTheInboxListsAlertsOfTheCarsAPersonCanOpenOnly(t *testing.T) {
	f, ts, _ := start(t)
	ann := login(t, ts.URL, "ann")
	bob := login(t, ts.URL, "bob")
	if code, _ := ann.do("POST", "/api/cars", `{"name":"Truck"}`, nil); code != 200 {
		t.Fatal("add car")
	}
	_ = f.Runtime(1).S.LogAlert(store.AlertLogEntry{TS: 100, Kind: "trip_end", Title: "Trip ended", Body: "5 mi", URL: "#/trip/1"})
	_ = f.Runtime(2).S.LogAlert(store.AlertLogEntry{TS: 200, Kind: "plug_in", Title: "Plugged in", URL: "#/charging"})
	var got struct {
		Items []struct {
			Car     int64  `json:"car"`
			CarName string `json:"carName"`
			Title   string `json:"title"`
			URL     string `json:"url"`
		} `json:"items"`
	}
	_, body := ann.do("GET", "/api/inbox", "", nil)
	if err := json.Unmarshal([]byte(body), &got); err != nil || len(got.Items) != 2 || got.Items[0].Title != "Plugged in" || got.Items[0].CarName != "Truck" || got.Items[1].URL != "#/trip/1" {
		t.Fatalf("an administrator with both cars sees both, newest first, with car names: %s", body)
	}
	_, body = bob.do("GET", "/api/inbox", "", nil)
	got.Items = nil
	_ = json.Unmarshal([]byte(body), &got)
	if len(got.Items) != 1 || got.Items[0].Car != 1 || got.Items[0].CarName != "" {
		t.Fatalf("a viewer of one car sees only its alerts, and no car name: %s", body)
	}
}

func TestTheDebugReportCanCoverThisCarOrEveryCarAndNamesNoCar(t *testing.T) {
	f, ts, _ := start(t)
	ann := login(t, ts.URL, "ann")
	if code, _ := ann.do("POST", "/api/cars", `{"name":"My secret truck name"}`, nil); code != 200 {
		t.Fatal("add car")
	}
	_, one := ann.do("GET", "/api/debug-report?hours=6", "", car("2"))
	if !strings.Contains(one, "installation: car 2 of 2 (added by a person)") || !strings.Contains(one, "categories: 3 (0 of your own)") || strings.Contains(one, "CAR 1 of") {
		t.Fatalf("this car's report says where it sits: %.600s", one)
	}
	if _, byQuery := ann.do("GET", "/api/debug-report?hours=6&car=2", "", nil); !strings.Contains(byQuery, "installation: car 2 of 2") {
		t.Fatalf("an administrator can pick any car by number: %.300s", byQuery)
	}
	_, all := ann.do("GET", "/api/debug-report?hours=6&all=1", "", nil)
	if !strings.Contains(all, "CAR 1 of 2") || !strings.Contains(all, "CAR 2 of 2") || strings.Contains(all, "My secret truck name") {
		t.Fatalf("every car, by number and never by name: %.600s", all)
	}
	// an owner who is not an administrator of the whole installation cannot ask for every car
	catU, _ := f.Core.UserByName("cat")
	_ = f.Core.SetOwnCar(catU.ID, true)
	cat := login(t, ts.URL, "cat")
	if code, _ := cat.do("POST", "/api/cars", `{"name":"c"}`, nil); code != 200 {
		t.Fatal("cat adds a car")
	}
	if code, _ := cat.do("GET", "/api/debug-report?hours=6&all=1", "", car("3")); code != 403 {
		t.Fatalf("every car is for the installation's administrators: %d", code)
	}
	if code, _ := cat.do("GET", "/api/debug-report?hours=6", "", car("3")); code != 200 {
		t.Fatalf("an owner gets her own car's report: %d", code)
	}
}
