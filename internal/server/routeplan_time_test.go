package server

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

// Roads are off, so the planner uses straight lines: three stops east of the start, then lunch to the west at a set time, and a
// last stop (the shop on the way home) that must stay last.
func TestLunchKeepsItsTimeWaitsWhenEarlyAndTheLastStopStaysLast(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	seedUsualDriving(t, srv)
	depart := time.Now().Add(24 * time.Hour).Truncate(time.Hour).Unix()
	lunch := depart + 3*3600
	body := fmt.Sprintf(`{"stops":[{"name":"Start","lat":45.0,"lon":-93.0},{"name":"East A","lat":45.0,"lon":-92.9},{"name":"Lunch","lat":45.0,"lon":-93.1,"atTs":%d,"stayMin":60},{"name":"East B","lat":45.0,"lon":-92.85},{"name":"Shop","lat":45.0,"lon":-93.02}],"departTs":%d,"startPct":90,"optimize":true,"roundTrip":false,"endLast":true,"stayMin":30}`, lunch, depart)
	code, out := do(t, admin, "POST", ts.URL+"/api/route-plan", body, xh)
	var r struct {
		Stops []struct{ Name string }
		Legs  []struct {
			To       string
			ArriveTS int64
			At       struct {
				AtTS                      int64
				WaitMin, LateMin, StayMin float64
			}
		}
		OnTime   bool
		Warnings []string
	}
	if code != 200 || json.Unmarshal([]byte(out), &r) != nil {
		t.Fatalf("%d %s", code, out)
	}
	if r.Stops[len(r.Stops)-1].Name != "Shop" {
		t.Fatalf("the shop stays last: %+v", r.Stops)
	}
	for _, l := range r.Legs {
		if l.To != "Lunch" {
			if l.At.StayMin != 30 {
				t.Fatalf("thirty minutes at each other stop: %+v", l)
			}
			continue
		}
		if l.At.AtTS != lunch || l.At.StayMin != 60 || l.At.LateMin > 10 || l.ArriveTS+int64(l.At.WaitMin*60) < lunch-60 {
			t.Fatalf("lunch at its time: %+v (lunch %d)", l, lunch)
		}
	}
	if !r.OnTime || len(r.Warnings) != 0 {
		t.Fatalf("on time, nothing to warn about: %v %v", r.OnTime, r.Warnings)
	}
	// Lunch ten minutes after leaving cannot be made after the other stops, and is not moved away from its time: it comes first.
	early := fmt.Sprintf(`{"stops":[{"name":"Start","lat":45.0,"lon":-93.0},{"name":"East A","lat":45.0,"lon":-92.9},{"name":"Lunch","lat":45.0,"lon":-93.1,"atTs":%d,"stayMin":60},{"name":"East B","lat":45.0,"lon":-92.85}],"departTs":%d,"startPct":90,"optimize":true,"stayMin":30}`, depart+600, depart)
	_, out = do(t, admin, "POST", ts.URL+"/api/route-plan", early, xh)
	if json.Unmarshal([]byte(out), &r) != nil || r.Stops[1].Name != "Lunch" {
		t.Fatalf("lunch first: %s", out)
	}
}
