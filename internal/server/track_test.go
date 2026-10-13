package server

import (
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestBrowserLoggerNeedsAnAdminAndStoresPoints(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	now := strconv.FormatInt(time.Now().Unix(), 10)
	body := `{"points":[{"ts":` + now + `,"lat":44.9,"lon":-93.2,"speed":2}]}`

	if code, _ := do(t, &http.Client{}, "POST", ts.URL+"/api/track/points", body, xh); code != 401 {
		t.Fatalf("anonymous: %d", code)
	}
	if code, _ := do(t, admin, "POST", ts.URL+"/api/track/points", body, nil); code != 403 {
		t.Fatalf("without the page header: %d", code)
	}
	if code, resp := do(t, admin, "POST", ts.URL+"/api/track/points", body, xh); code != 200 || !strings.Contains(resp, `"saved":1`) {
		t.Fatalf("admin logging: %d %s", code, resp)
	}
	do(t, admin, "POST", ts.URL+"/api/track/points", body, xh) // a second call reuses the same stand-in device
	ds, _ := srv.S.ListDevices()
	if len(ds) != 1 || !strings.HasPrefix(ds[0].Name, "Browser (owner)") || ds[0].Points != 2 {
		t.Fatalf("devices: %+v", ds)
	}
	if srv.P.PhoneActive() {
		t.Fatal("2 m/s is walking pace and must not look like a trip")
	}

	do(t, &http.Client{}, "POST", ts.URL+"/api/ingest/points", body, bearer("browser:1"))
	if ps, _ := srv.S.PointsBetween(0, time.Now().Unix()+60); len(ps) != 2 {
		t.Fatalf("the stand-in device must not be usable as a token: %d points", len(ps))
	}
}

func TestFastPointsLookLikeADrive(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	_, tok := newDevice(t, ts.URL, admin, "Phone")
	now := time.Now().Unix()
	old := `{"points":[{"ts":` + strconv.FormatInt(now-3600, 10) + `,"lat":44.9,"lon":-93.2,"speed":20}]}`
	do(t, &http.Client{}, "POST", ts.URL+"/api/ingest/points", old, bearer(tok))
	if srv.P.PhoneActive() {
		t.Fatal("an hour-old queued fix must not start the fast pace")
	}
	fresh := `{"points":[{"ts":` + strconv.FormatInt(now, 10) + `,"lat":44.9,"lon":-93.2,"speed":20}]}`
	do(t, &http.Client{}, "POST", ts.URL+"/api/ingest/points", fresh, bearer(tok))
	if !srv.P.PhoneActive() {
		t.Fatal("a fresh 72 km/h fix should tell the poller a trip is under way")
	}
}

func TestStatusShowsPhoneTracking(t *testing.T) {
	ts, _ := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	_, body := do(t, admin, "GET", ts.URL+"/api/health", "", nil)
	if !strings.Contains(body, `"id":"phone"`) || !strings.Contains(body, `"href":"#/phone"`) {
		t.Fatalf("no phone line: %s", body)
	}
	_, tok := newDevice(t, ts.URL, admin, "Pixel")
	_, body = do(t, admin, "GET", ts.URL+"/api/health", "", nil)
	if !strings.Contains(body, "has not sent anything yet") {
		t.Fatalf("paired but silent: %s", body)
	}
	do(t, &http.Client{}, "POST", ts.URL+"/api/ingest/event", `{"type":"ping"}`, bearer(tok))
	_, body = do(t, admin, "GET", ts.URL+"/api/health", "", nil)
	if !strings.Contains(body, "Pixel last heard from just now") || strings.Contains(body, tok) {
		t.Fatalf("heard from: %s", body)
	}
	_, list := do(t, admin, "GET", ts.URL+"/api/devices", "", nil)
	if !strings.Contains(list, `"address":"http://`) {
		t.Fatalf("the page needs the address to build the recipes: %s", list)
	}
}
