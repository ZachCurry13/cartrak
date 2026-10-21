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
