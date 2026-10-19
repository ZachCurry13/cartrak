package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
	"github.com/ZachCurry13/cartrak/internal/trips"
)

func TestVacationMakesTripsPersonalAndWorkReturnsWhenItIsRemoved(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	srv.D = &trips.Detector{S: srv.S, TZ: time.UTC}
	admin := adminClient(t, ts.URL)
	srv.S.SaveRule(store.Rule{Name: "Work hours", Category: trips.Work, Enabled: true, Days: 127, StartMin: 0, EndMin: 1440})
	mk := func(day time.Time) int64 {
		end := day.Add(20 * time.Minute).Unix()
		tr := &store.Trip{StartTS: day.Unix(), EndTS: &end, DistanceKm: 10, Category: "work", CategorySource: "rule", TZ: "UTC"}
		if err := srv.S.CreateTrip(tr); err != nil {
			t.Fatal(err)
		}
		tr.EndTS, tr.Open = &end, false
		srv.S.UpdateTrip(tr)
		return tr.ID
	}
	in := time.Date(2026, 10, 21, 10, 0, 0, 0, time.UTC)
	out := time.Date(2026, 10, 30, 10, 0, 0, 0, time.UTC)
	a, b := mk(in), mk(out)
	cat := func(id int64) string {
		tr, _ := srv.S.GetTrip(id)
		return tr.Category + "/" + tr.CategorySource
	}
	code, body := do(t, admin, "POST", ts.URL+"/api/time-off", `{"from":"2026-10-19","to":"2026-10-26","label":"Beach"}`, xh)
	var made store.TimeOff
	if code != 200 || json.Unmarshal([]byte(body), &made) != nil {
		t.Fatalf("%d %s", code, body)
	}
	if cat(a) != "personal/vacation" || cat(b) != "work/rule" {
		t.Fatalf("trips inside the vacation become personal, outside stay: %s %s", cat(a), cat(b))
	}
	// A trip chosen by hand keeps its choice.
	if err := srv.S.SetTripCategory(a, "work", "client"); err != nil {
		t.Fatal(err)
	}
	srv.D.Reclassify()
	if cat(a) != "work/manual" {
		t.Fatalf("the owner's own choice wins: %s", cat(a))
	}
	// New trips are personal as they are recorded.
	if cls, _, src := classifyForTest(srv, in.Add(24*time.Hour)); cls != "personal" || src != "vacation" {
		t.Fatalf("%s %s", cls, src)
	}
	if code, _ := do(t, admin, "POST", ts.URL+"/api/time-off", `{"from":"2026-10-26","to":"2026-10-19"}`, xh); code != 400 {
		t.Fatalf("backwards dates: %d", code)
	}
	if code, _ := do(t, admin, "DELETE", fmt.Sprintf("%s/api/time-off/%d", ts.URL, made.ID), "", xh); code != 200 {
		t.Fatalf("delete: %d", code)
	}
	srv.S.SetTripCategory(a, "personal", "")
	do(t, admin, "POST", ts.URL+"/api/rules/apply", "", xh)
	if cat(b) != "work/rule" {
		t.Fatalf("after the vacation is gone the rules apply again: %s", cat(b))
	}
	do(t, admin, "POST", ts.URL+"/api/users", `{"name":"fam","password":"viewer-password-1","role":"viewer"}`, xh)
	vj, _ := cookieJar()
	v := &http.Client{Jar: vj}
	do(t, v, "POST", ts.URL+"/api/login", `{"name":"fam","password":"viewer-password-1"}`, nil)
	if code, _ := do(t, v, "POST", ts.URL+"/api/time-off", `{"from":"2026-10-19","to":"2026-10-20"}`, xh); code != 403 {
		t.Fatalf("a viewer cannot set a vacation: %d", code)
	}
}

// classifyForTest runs the detector's classification for a trip starting at ts.
func classifyForTest(srv *Server, at time.Time) (string, *int64, string) {
	tr := &store.Trip{StartTS: at.Unix(), TZ: "UTC"}
	srv.D.ClassifyTrip(tr)
	return tr.Category, tr.RuleID, tr.CategorySource
}
