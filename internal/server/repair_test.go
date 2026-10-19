package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/store"
	"github.com/ZachCurry13/cartrak/internal/trips"
)

func TestRepairPreviewShowsTheChangeApplyNeedsTheWordAndViewersAreRefused(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	srv.D = &trips.Detector{S: srv.S, TZ: time.UTC}
	admin := adminClient(t, ts.URL)
	t0 := time.Now().Add(-30 * time.Hour).Truncate(time.Minute)
	fp := func(v float64) *float64 { return &v }
	for i, x := range []struct {
		min      int
		odo, kph float64
		ign      string
	}{{0, 1000, 80, "ON"}, {30, 1040, 80, "ON"}, {40, 1040, 0, "OFF"}} {
		st := &ford.State{TS: t0.Add(time.Duration(x.min) * time.Minute), OdometerKm: fp(x.odo), SoCPct: fp(80 - float64(i)), SpeedKph: fp(x.kph), Ignition: x.ign, Raw: []byte(`{}`)}
		if _, err := srv.S.SaveState("t", st); err != nil {
			t.Fatal(err)
		}
	}
	body := func(extra string) string {
		return fmt.Sprintf(`{"from":%d,"to":%d%s}`, t0.Unix(), t0.Add(40*time.Minute).Unix(), extra)
	}
	code, b := do(t, admin, "POST", ts.URL+"/api/repair/preview", body(""), xh)
	var plan struct {
		NewTrips []store.Trip
		Blockers []string
	}
	if code != 200 || json.Unmarshal([]byte(b), &plan) != nil || len(plan.NewTrips) != 1 || len(plan.Blockers) != 0 {
		t.Fatalf("preview: %d %s", code, b)
	}
	if left, _ := srv.S.ListTrips(store.TripFilter{}); len(left) != 0 {
		t.Fatal("a preview changes nothing")
	}
	if code, _ := do(t, admin, "POST", ts.URL+"/api/repair/apply", body(""), xh); code != 400 {
		t.Fatalf("apply without the word: %d", code)
	}
	if code, b := do(t, admin, "POST", ts.URL+"/api/repair/apply", body(`,"confirm":"REBUILD"`), xh); code != 200 {
		t.Fatalf("apply: %d %s", code, b)
	}
	if got, _ := srv.S.ListTrips(store.TripFilter{}); len(got) != 1 || got[0].DistanceKm < 39 || got[0].DistanceKm > 41 {
		t.Fatalf("the trip was rebuilt from the readings: %+v", got)
	}
	do(t, admin, "POST", ts.URL+"/api/users", `{"name":"fam","password":"viewer-password-1","role":"viewer"}`, xh)
	vj, _ := cookieJar()
	v := &http.Client{Jar: vj}
	do(t, v, "POST", ts.URL+"/api/login", `{"name":"fam","password":"viewer-password-1"}`, nil)
	if code, _ := do(t, v, "POST", ts.URL+"/api/repair/preview", body(""), xh); code != 403 {
		t.Fatalf("viewer: %d", code)
	}
}

func TestRepairPreviewAlwaysReturnsListsNeverNull(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	srv.D = &trips.Detector{S: srv.S, TZ: time.UTC}
	admin := adminClient(t, ts.URL)
	now := time.Now()
	code, b := do(t, admin, "POST", ts.URL+"/api/repair/preview", fmt.Sprintf(`{"from":%d,"to":%d}`, now.Unix(), now.Add(-time.Hour).Unix()), xh)
	if code != 200 {
		t.Fatalf("%d %s", code, b)
	}
	for _, want := range []string{`"blockers":["`, `"oldTrips":[]`, `"newTrips":[]`, `"oldCharges":[]`, `"newCharges":[]`} {
		if !strings.Contains(b, want) {
			t.Errorf("missing %s in %s", want, b)
		}
	}
}
