package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/ZachCurry13/cartrak/internal/trips"
)

func TestSetupChecklistIsPerPersonAndCanBeHidden(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	get := func() map[string]any {
		code, body := do(t, admin, "GET", ts.URL+"/api/checklist", "", nil)
		var m map[string]any
		if code != 200 || json.Unmarshal([]byte(body), &m) != nil {
			t.Fatalf("checklist: %d %s", code, body)
		}
		return m
	}
	m := get()
	for _, k := range []string{"home", "price", "gas", "placePrice", "phone", "maintenance", "backup", "alerts"} {
		if _, ok := m[k]; !ok {
			t.Fatalf("an admin sees %s: %v", k, m)
		}
	}
	if m["hidden"] != false || m["redetect"] != false {
		t.Fatalf("shown, and no Fix a day offer on a new installation: %v", m)
	}
	if code, _ := do(t, admin, "POST", ts.URL+"/api/checklist/hide", "", xh); code != 200 || get()["hidden"] != true {
		t.Fatal("hidden")
	}
	if code, _ := do(t, admin, "POST", ts.URL+"/api/checklist/hide?show=1", "", xh); code != 200 || get()["hidden"] != false {
		t.Fatal("shown again")
	}
	do(t, admin, "POST", ts.URL+"/api/checklist/skip?key=gas", "", xh)
	do(t, admin, "POST", ts.URL+"/api/checklist/skip?key=tour", "", xh)
	do(t, admin, "POST", ts.URL+"/api/checklist/skip?key=gas&undo=1", "", xh)
	if sk, _ := get()["skipped"].([]any); len(sk) != 1 || sk[0] != "tour" {
		t.Fatalf("skipped, and one brought back: %v", get()["skipped"])
	}
	if code, _ := do(t, admin, "POST", ts.URL+"/api/checklist/skip?key=../x", "", xh); code != 400 {
		t.Fatal("only plain keys")
	}
	// after an update that finds trips differently, the offer appears until it is put away
	_ = srv.S.SetKV(kvDetectionSeen, strconv.Itoa(trips.DetectionVersion-1))
	if get()["redetect"] != true {
		t.Fatal("offered after a detection change")
	}
	if code, _ := do(t, admin, "POST", ts.URL+"/api/repair/seen", "", xh); code != 200 || get()["redetect"] != false {
		t.Fatal("put away")
	}
	if code, _ := do(t, &http.Client{}, "GET", ts.URL+"/api/checklist", "", nil); code == 200 {
		t.Fatal("signed-out callers get nothing")
	}
}

func TestMeCarriesTheCarsBatteryReserve(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	if _, body := do(t, admin, "GET", ts.URL+"/api/me", "", nil); !strings.Contains(body, `"reservePct":20`) {
		t.Fatalf("20%% unless chosen: %s", body)
	}
	_ = srv.Cfg.KV.SetKV("setting:CARTRAK_LOW_BATTERY", "30")
	if _, body := do(t, admin, "GET", ts.URL+"/api/me", "", nil); !strings.Contains(body, `"reservePct":30`) || !strings.Contains(body, `"role":"admin"`) {
		t.Fatalf("the chosen reserve, with who is signed in: %s", body)
	}
}

func TestEachPersonKeepsTheirOwnBarAndTiles(t *testing.T) {
	ts, _ := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	if _, body := do(t, admin, "GET", ts.URL+"/api/me/layout", "", nil); body != "{}\n" {
		t.Fatalf("nothing chosen yet: %q", body)
	}
	if code, _ := do(t, admin, "PUT", ts.URL+"/api/me/layout", `{"bar":["home","insights","trips"],"tiles":["reports"],"auto":true}`, xh); code != 200 {
		t.Fatal("save")
	}
	if _, body := do(t, admin, "GET", ts.URL+"/api/me/layout", "", nil); !strings.Contains(body, `"bar":["home","insights","trips"]`) || !strings.Contains(body, `"auto":true`) {
		t.Fatalf("kept: %s", body)
	}
	for _, bad := range []string{`{"bar":[],"tiles":[]}`, `{"bar":["a","b","c","d","e","f","g"],"tiles":[]}`, `{"bar":["../x"],"tiles":[]}`} {
		if code, _ := do(t, admin, "PUT", ts.URL+"/api/me/layout", bad, xh); code != 400 {
			t.Fatalf("%s should be refused", bad)
		}
	}
}
