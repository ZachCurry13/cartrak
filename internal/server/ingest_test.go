package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

// newDevice has the admin create a device and returns its token.
func newDevice(t *testing.T, base string, admin *http.Client, name string) (int64, string) {
	code, body := do(t, admin, "POST", base+"/api/devices", `{"name":"`+name+`"}`, xh)
	if code != 200 {
		t.Fatalf("create device = %d %s", code, body)
	}
	var out struct {
		ID    int64  `json:"id"`
		Token string `json:"token"`
	}
	json.Unmarshal([]byte(body), &out)
	return out.ID, out.Token
}

func bearer(tok string) map[string]string { return map[string]string{"Authorization": "Bearer " + tok} }

func TestDeviceTokenIsShownOnceAndOnlyAdminsManageDevices(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	_, tok := newDevice(t, ts.URL, admin, "My phone")
	if !strings.HasPrefix(tok, "ct_") || len(tok) < 40 {
		t.Fatalf("token looks weak: %q", tok)
	}
	_, list := do(t, admin, "GET", ts.URL+"/api/devices", "", nil)
	if strings.Contains(list, tok) || !strings.Contains(list, "My phone") {
		t.Fatalf("the list must name the device but never show the token: %s", list)
	}
	var stored string
	srv.S.DB.QueryRow(`SELECT token_hash FROM devices`).Scan(&stored)
	if strings.Contains(stored, tok) {
		t.Fatal("token stored in clear")
	}
	do(t, admin, "POST", ts.URL+"/api/users", `{"name":"fam","password":"viewer-password-1","role":"viewer"}`, xh)
	vj, _ := cookieJar()
	v := &http.Client{Jar: vj}
	do(t, v, "POST", ts.URL+"/api/login", `{"name":"fam","password":"viewer-password-1"}`, nil)
	if code, _ := do(t, v, "POST", ts.URL+"/api/devices", `{"name":"x"}`, xh); code != 403 {
		t.Fatalf("a viewer made a device: %d", code)
	}
	if code, _ := do(t, v, "GET", ts.URL+"/api/devices", "", nil); code != 403 {
		t.Fatalf("a viewer read devices: %d", code)
	}
	if code, _ := do(t, admin, "POST", ts.URL+"/api/devices", `{"name":"  "}`, xh); code != 400 {
		t.Fatalf("empty name accepted: %d", code)
	}
}

func TestIngestEventStoresItAndWakesThePoller(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	_, tok := newDevice(t, ts.URL, admin, "Phone")
	c := &http.Client{}

	if code, b := do(t, c, "POST", ts.URL+"/api/ingest/event", `{"type":"drive_start","lat":44.98,"lon":-93.26,"acc":12}`, bearer(tok)); code != 200 {
		t.Fatalf("event = %d %s", code, b)
	}
	if !srv.P.PhoneActive() {
		t.Fatal("drive_start should tell the poller a trip is under way")
	}
	es, _ := srv.S.RecentEvents(5)
	if len(es) != 1 || es[0].Type != "drive_start" || es[0].Lat == nil || *es[0].Lat != 44.98 {
		t.Fatalf("events: %+v", es)
	}
	if ps, _ := srv.S.PointsBetween(0, time.Now().Unix()+60); len(ps) != 1 {
		t.Fatalf("the start location should be kept as a point: %+v", ps)
	}
	if ds, _ := srv.S.ListDevices(); ds[0].LastEvent != "drive_start" || ds[0].LastSeen == 0 {
		t.Fatalf("device not marked seen: %+v", ds)
	}
	do(t, c, "POST", ts.URL+"/api/ingest/event", `{"type":"drive_end"}`, bearer(tok))
	if srv.P.PhoneActive() {
		t.Fatal("drive_end should clear the hint")
	}
	do(t, c, "POST", ts.URL+"/api/ingest/event", `{"type":"ping"}`, bearer(tok))
	if srv.P.PhoneActive() {
		t.Fatal("a ping must not start a trip")
	}
	for body, why := range map[string]string{`{"type":"dance"}`: "unknown type", `{"type":"ping","ts":5}`: "ancient time", `nope`: "not json"} {
		if code, _ := do(t, c, "POST", ts.URL+"/api/ingest/event", body, bearer(tok)); code != 400 {
			t.Fatalf("%s should be 400, got %d", why, code)
		}
	}
}

func TestIngestRejectsMissingWrongAndRemovedTokens(t *testing.T) {
	ts, _ := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	id, tok := newDevice(t, ts.URL, admin, "Phone")
	c := &http.Client{}
	post := func(h map[string]string) int {
		code, _ := do(t, c, "POST", ts.URL+"/api/ingest/points", `{"points":[]}`, h)
		return code
	}
	if post(nil) != 401 || post(bearer("ct_wrong")) != 401 || post(map[string]string{"Authorization": tok}) != 401 {
		t.Fatal("a missing, wrong or malformed token was accepted")
	}
	if code, _ := do(t, admin, "POST", ts.URL+"/api/ingest/points", `{"points":[]}`, xh); code != 401 {
		t.Fatalf("a login session must not work as a device: %d", code)
	}
	if post(bearer(tok)) != 200 {
		t.Fatal("the real token was refused")
	}
	do(t, admin, "DELETE", ts.URL+"/api/devices/"+strconv.FormatInt(id, 10), "", xh)
	if post(bearer(tok)) != 401 {
		t.Fatal("a removed device's token still works")
	}
}

func TestWrongTokensLockTheAddressOut(t *testing.T) {
	ts, _ := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	_, tok := newDevice(t, ts.URL, admin, "Phone")
	c := &http.Client{}
	for i := 0; i < 5; i++ {
		do(t, c, "POST", ts.URL+"/api/ingest/points", `{}`, bearer("ct_guess"+strconv.Itoa(i)))
	}
	if code, _ := do(t, c, "POST", ts.URL+"/api/ingest/points", `{}`, bearer(tok)); code != 429 {
		t.Fatalf("after 5 wrong tokens even the right one should wait: %d", code)
	}
}

func TestIngestPointsAreCheckedAndConverted(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	_, tok := newDevice(t, ts.URL, admin, "Phone")
	c := &http.Client{}
	now := time.Now()
	body := `{"points":[
	  {"ts":` + strconv.FormatInt(now.Unix()-30, 10) + `,"lat":"44.9","lon":"-93.2","speed":"10","acc":5},
	  {"ts":` + strconv.FormatInt(now.UnixMilli()-20000, 10) + `,"lat":44.91,"lon":-93.21,"speed":-1},
	  {"lat":95,"lon":0},
	  {"lat":0,"lon":0},
	  {"lat":44.9},
	  {"ts":` + strconv.FormatInt(now.Add(time.Hour).Unix(), 10) + `,"lat":44.9,"lon":-93.2},
	  {"ts":1000,"lat":44.9,"lon":-93.2}]}`
	code, resp := do(t, c, "POST", ts.URL+"/api/ingest/points", body, bearer(tok))
	if code != 200 || !strings.Contains(resp, `"saved":2`) || !strings.Contains(resp, `"skipped":5`) {
		t.Fatalf("points = %d %s", code, resp)
	}
	ps, _ := srv.S.PointsBetween(now.Unix()-60, now.Unix()+5)
	if len(ps) != 2 {
		t.Fatalf("stored: %+v", ps)
	}
	if ps[0].SpeedKph == nil || *ps[0].SpeedKph < 35.9 || *ps[0].SpeedKph > 36.1 {
		t.Fatalf("10 m/s should be 36 km/h: %+v", ps[0])
	}
	if ps[1].SpeedKph != nil {
		t.Fatalf("an unknown speed (-1) must stay empty: %+v", ps[1])
	}
	many := `{"points":[` + strings.Repeat(`{"lat":1,"lon":1},`, maxPointsPerCall) + `{"lat":1,"lon":1}]}`
	if code, _ := do(t, c, "POST", ts.URL+"/api/ingest/points", many, bearer(tok)); code != 400 {
		t.Fatalf("an oversized batch should be refused: %d", code)
	}
}

func TestDeviceRateLimit(t *testing.T) {
	var r rateWindow
	for i := 0; i < callsPerMinute; i++ {
		if !r.allow(1) {
			t.Fatalf("call %d refused too early", i)
		}
	}
	if r.allow(1) {
		t.Fatal("the next call in the same minute should be refused")
	}
	if !r.allow(2) {
		t.Fatal("another device has its own allowance")
	}
}
