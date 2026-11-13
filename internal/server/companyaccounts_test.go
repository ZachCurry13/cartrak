package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/settings"
)

func TestAccountsAreListedGuessedByNameAndForgotten(t *testing.T) {
	svc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`[]`)) }))
	defer svc.Close()
	ts, srv := healthServer(t, fakeProv{}, false, false)
	srv.GeoGap = time.Millisecond
	srv.Cfg.Set(settings.GeocodeURLKey, svc.URL)
	admin := adminClient(t, ts.URL)
	if c, _ := do(t, admin, "POST", ts.URL+"/api/company-list/remember", `{"ref":"555","name":"Mystery Metals","address":"9 Oak St","lat":45.1,"lon":-93.3}`, xh); c != 200 {
		t.Fatalf("remember: %d", c)
	}
	code, body := do(t, admin, "GET", ts.URL+"/api/company-list/accounts", "", nil)
	var l []struct{ Ref, Name, Address string }
	if code != 200 || json.Unmarshal([]byte(body), &l) != nil || len(l) != 1 || l[0].Ref != "555" || l[0].Address != "9 Oak St" {
		t.Fatalf("list: %d %s", code, body)
	}
	b, _ := json.Marshal(map[string]string{"text": "Mystery Metals, Inc.-999-U.S.A.\nMystery Metals LLC"})
	code, body = do(t, admin, "POST", ts.URL+"/api/company-list/resolve", string(b), xh)
	var r struct {
		Rows []struct {
			Status string
			Known  bool
			Pick   *struct{ Address string }
		}
	}
	if code != 200 || json.Unmarshal([]byte(body), &r) != nil || len(r.Rows) != 2 {
		t.Fatalf("resolve: %d %s", code, body)
	}
	for i, x := range r.Rows {
		if x.Status != "remembered" || x.Known || x.Pick == nil || x.Pick.Address != "9 Oak St" {
			t.Fatalf("row %d is guessed by its name: %+v", i, x)
		}
	}
	if c, _ := do(t, admin, "DELETE", ts.URL+"/api/company-list/accounts/abc", "", xh); c != 400 {
		t.Fatalf("a non-number is refused, got %d", c)
	}
	if c, _ := do(t, admin, "DELETE", ts.URL+"/api/company-list/accounts/555", "", xh); c != 200 {
		t.Fatalf("forget: %d", c)
	}
	if _, body = do(t, admin, "GET", ts.URL+"/api/company-list/accounts", "", nil); strings.TrimSpace(body) != "[]" {
		t.Fatalf("forgotten: %s", body)
	}
}

func TestTwoPlacesUnderOneNameAreNotGuessed(t *testing.T) {
	list := []account{{Ref: "1", companyPick: companyPick{Name: "Acme Widgets", Lat: 1, Lon: 1}}, {Ref: "2", companyPick: companyPick{Name: "Acme Widgets", Lat: 2, Lon: 2}}}
	if byName(list, "Acme Widgets Inc") != nil {
		t.Fatal("two branches of one company must be asked about")
	}
	if a := byName(list[:1], "ACME WIDGETS, INC."); a == nil || a.Ref != "1" {
		t.Fatalf("one match is taken: %+v", a)
	}
}

func TestADaysSavedRouteIsClearedAfterTwoWeeksAndALongTripIsKept(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	old := time.Now().Add(-15 * 24 * time.Hour).Unix()
	if _, err := srv.S.AddSavedTrip("Old route", `{"kind":"route","stops":[]}`, old); err != nil {
		t.Fatal(err)
	}
	if _, err := srv.S.AddSavedTrip("Old long trip", `{"body":{},"result":{}}`, old); err != nil {
		t.Fatal(err)
	}
	if c, _ := do(t, admin, "POST", ts.URL+"/api/saved-trips", `{"name":"Thursday","body":{"kind":"route","stops":[{"name":"A","lat":1,"lon":1},{"name":"B","lat":2,"lon":2}]}}`, xh); c != 200 {
		t.Fatalf("save: %d", c)
	}
	_, body := do(t, admin, "GET", ts.URL+"/api/saved-trips", "", nil)
	if strings.Contains(body, "Old route") || !strings.Contains(body, "Old long trip") || !strings.Contains(body, "Thursday") {
		t.Fatalf("%s", body)
	}
	if all, _ := srv.S.SavedTrips(); len(all) != 2 {
		t.Fatalf("the old route is deleted, not only hidden: %d", len(all))
	}
}
