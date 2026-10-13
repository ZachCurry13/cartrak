package server

import (
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

func basic(user, token string) map[string]string {
	return map[string]string{"Authorization": "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+token))}
}

func TestOwnTracksPostsAreAcceptedWithBasicLogin(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	_, tok := newDevice(t, ts.URL, admin, "Phone")
	c := &http.Client{}
	now := time.Now().Unix()
	post := func(h map[string]string, body string) (int, string) {
		return do(t, c, "POST", ts.URL+"/api/ingest/owntracks", body, h)
	}

	code, resp := post(basic("anyone", tok), `{"_type":"location","tid":"ab","lat":10.5,"lon":20.5,"tst":`+strconv.FormatInt(now, 10)+`,"acc":12,"vel":54,"batt":80}`)
	if code != 200 || strings.TrimSpace(resp) != "[]" {
		t.Fatalf("OwnTracks expects a 200 with an empty array: %d %q", code, resp)
	}
	ps, _ := srv.S.PointsBetween(now-5, now+5)
	if len(ps) != 1 || ps[0].Lat != 10.5 || ps[0].SpeedKph == nil || *ps[0].SpeedKph != 54 || ps[0].AccM == nil || *ps[0].AccM != 12 {
		t.Fatalf("the location should be stored with speed in km/h as given: %+v", ps)
	}
	if !srv.P.PhoneActive() {
		t.Fatal("54 km/h counts as driving for the faster checks")
	}

	// Everything else is acknowledged and ignored; nothing is stored.
	for name, body := range map[string]string{
		"a waypoint":  `{"_type":"waypoint","desc":"Home","lat":1,"lon":2,"tst":` + strconv.FormatInt(now, 10) + `}`,
		"empty":       ``,
		"junk":        `not json`,
		"no position": `{"_type":"location","tst":` + strconv.FormatInt(now, 10) + `}`,
		"ancient":     `{"_type":"location","lat":1,"lon":2,"tst":5}`,
	} {
		if code, resp := post(basic("x", tok), body); code != 200 || strings.TrimSpace(resp) != "[]" {
			t.Errorf("%s: %d %q", name, code, resp)
		}
	}
	if ps, _ := srv.S.PointsBetween(0, now+10); len(ps) != 1 {
		t.Fatalf("only the real location may be stored: %d points", len(ps))
	}
	if ds, _ := srv.S.ListDevices(); ds[0].LastSeen == 0 {
		t.Fatal("the phone should show as heard from")
	}
}

func TestOwnTracksNeedsTheRightToken(t *testing.T) {
	ts, _ := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	_, tok := newDevice(t, ts.URL, admin, "Phone")
	c := &http.Client{}
	body := `{"_type":"location","lat":1,"lon":2}`
	for name, h := range map[string]map[string]string{"no login": nil, "wrong password": basic("u", "ct_wrong"), "token as user name": basic(tok, "x"), "garbage": {"Authorization": "Basic !!!"}} {
		if code, _ := do(t, c, "POST", ts.URL+"/api/ingest/owntracks", body, h); code != 401 {
			t.Errorf("%s should be refused, got %d", name, code)
		}
	}
	// The existing Bearer form still works on every ingest address.
	if code, _ := do(t, c, "POST", ts.URL+"/api/ingest/points", `{"points":[]}`, bearer(tok)); code != 200 {
		t.Fatalf("bearer must keep working: %d", code)
	}
	if code, _ := do(t, c, "POST", ts.URL+"/api/ingest/points", `{"points":[]}`, basic("u", tok)); code != 200 {
		t.Fatalf("basic works on the other ingest addresses too: %d", code)
	}
}
