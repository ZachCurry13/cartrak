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
)

func addStopTrip(t *testing.T, srv *Server, start time.Time, minutes int, lat, lon float64, cat string) *store.Trip {
	end := start.Add(time.Duration(minutes) * time.Minute).Unix()
	tr := &store.Trip{StartTS: start.Unix(), Category: cat, CategorySource: "manual"}
	if err := srv.S.CreateTrip(tr); err != nil {
		t.Fatal(err)
	}
	tr.EndTS, tr.EndLat, tr.EndLon, tr.DistanceKm, tr.Open = &end, &lat, &lon, 10, false
	if err := srv.S.UpdateTrip(tr); err != nil {
		t.Fatal(err)
	}
	return tr
}

func TestWorkStopsAreTheParkedGapsAfterTripsAndCanBeLabelled(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	t0 := time.Now().Add(-5 * time.Hour).Truncate(time.Minute)
	a := addStopTrip(t, srv, t0, 20, 12.3, 23.4, "work")                // arrives at a customer
	addStopTrip(t, srv, t0.Add(80*time.Minute), 15, 12.9, 23.9, "work") // leaves 60 minutes later

	var v struct {
		Stops []struct {
			TripID   int64  `json:"tripId"`
			Minutes  int    `json:"minutes"`
			Category string `json:"category"`
			Manual   bool   `json:"manual"`
			Note     string `json:"note"`
			Open     bool   `json:"open"`
		} `json:"stops"`
		WorkMinutes int `json:"workMinutes"`
	}
	get := func(cl *http.Client) string {
		_, body := do(t, cl, "GET", ts.URL+"/api/stops", "", nil)
		v.Stops = nil
		if err := json.Unmarshal([]byte(body), &v); err != nil {
			t.Fatal(err)
		}
		return body
	}
	body := get(admin)
	if len(v.Stops) != 2 || v.Stops[1].TripID != a.ID || v.Stops[1].Minutes != 60 || v.Stops[1].Category != "work" || v.Stops[1].Manual {
		t.Fatalf("an hour in the customer's lot after a work trip is a work stop (oldest last): %s", body)
	}
	if !v.Stops[0].Open || v.WorkMinutes < 60 {
		t.Fatalf("the car is still parked after the last trip, and the work minutes add up: %s", body)
	}

	if code, b := do(t, admin, "PUT", fmt.Sprintf("%s/api/stops/%d", ts.URL, a.ID), `{"category":"personal","note":"Acme Corp"}`, xh); code != 200 {
		t.Fatalf("label: %d %s", code, b)
	}
	get(admin)
	if v.Stops[1].Category != "personal" || !v.Stops[1].Manual || v.Stops[1].Note != "Acme Corp" {
		t.Fatalf("the label wins: %+v", v.Stops[1])
	}
	for _, bad := range []string{`{"category":"nonsense"}`, `{"note":"` + strings.Repeat("x", 201) + `"}`} {
		if code, _ := do(t, admin, "PUT", fmt.Sprintf("%s/api/stops/%d", ts.URL, a.ID), bad, xh); code != 400 {
			t.Fatalf("%s should be refused", bad)
		}
	}
	if code, _ := do(t, admin, "PUT", ts.URL+"/api/stops/99999", `{"note":"x"}`, xh); code != 404 {
		t.Fatalf("a stop that does not exist: %d", code)
	}
	do(t, admin, "PUT", fmt.Sprintf("%s/api/stops/%d", ts.URL, a.ID), `{"category":"","note":""}`, xh)
	get(admin)
	if v.Stops[1].Category != "work" || v.Stops[1].Manual {
		t.Fatalf("clearing the label returns to the trip's category: %+v", v.Stops[1])
	}

	// a viewer can read the stops but not change them
	do(t, admin, "POST", ts.URL+"/api/users", `{"name":"fam","password":"viewer-password-1","role":"viewer"}`, xh)
	vj, _ := cookieJar()
	viewer := &http.Client{Jar: vj}
	do(t, viewer, "POST", ts.URL+"/api/login", `{"name":"fam","password":"viewer-password-1"}`, nil)
	if code, _ := do(t, viewer, "GET", ts.URL+"/api/stops", "", nil); code != 200 {
		t.Fatalf("viewer read = %d", code)
	}
	if code, _ := do(t, viewer, "PUT", fmt.Sprintf("%s/api/stops/%d", ts.URL, a.ID), `{"note":"x"}`, xh); code != 403 {
		t.Fatalf("viewer label = %d", code)
	}
}

func TestStopsCsvHasTheNoteButNeverAFormula(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	t0 := time.Now().Add(-5 * time.Hour).Truncate(time.Minute)
	a := addStopTrip(t, srv, t0, 20, 12.3, 23.4, "work")
	addStopTrip(t, srv, t0.Add(80*time.Minute), 15, 12.9, 23.9, "work")
	do(t, admin, "PUT", fmt.Sprintf("%s/api/stops/%d", ts.URL, a.ID), `{"note":"=HYPERLINK(\"x\")"}`, xh)
	code, body := do(t, admin, "GET", ts.URL+"/api/stops.csv", "", nil)
	if code != 200 || !strings.HasPrefix(body, "Date,Start,End,Minutes,Place,Category,Note") || !strings.Contains(body, ",60,") {
		t.Fatalf("csv: %d %q", code, body)
	}
	if strings.Contains(body, ",=HYPERLINK") || !strings.Contains(body, "'=HYPERLINK") {
		t.Fatalf("a note that starts with = must not be a formula: %q", body)
	}
}

func TestAStopBeginsWhenTheCarStoppedMovingNotWhenTheEngineWasSwitchedOff(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	t0 := time.Now().Add(-5 * time.Hour).Truncate(time.Minute)
	a := addStopTrip(t, srv, t0, 20, 12.3, 23.4, "work") // moving until minute 8, then sitting with the engine on until minute 20
	addStopTrip(t, srv, t0.Add(50*time.Minute), 10, 12.9, 23.9, "work")
	for m := 0; m <= 20; m += 2 {
		sp := 60.0
		if m > 8 {
			sp = 0
		}
		odo := 1000.0 + float64(m)
		if _, err := srv.S.SaveState("t", &ford.State{TS: t0.Add(time.Duration(m) * time.Minute), SpeedKph: &sp, OdometerKm: &odo, Raw: []byte(`{}`)}); err != nil {
			t.Fatal(err)
		}
	}
	_, body := do(t, admin, "GET", ts.URL+"/api/stops", "", nil)
	var v struct {
		Stops []struct {
			TripID  int64 `json:"tripId"`
			Start   int64 `json:"start"`
			Minutes int   `json:"minutes"`
		} `json:"stops"`
	}
	if err := json.Unmarshal([]byte(body), &v); err != nil {
		t.Fatal(err)
	}
	var got *struct {
		TripID  int64 `json:"tripId"`
		Start   int64 `json:"start"`
		Minutes int   `json:"minutes"`
	}
	for i := range v.Stops {
		if v.Stops[i].TripID == a.ID {
			got = &v.Stops[i]
		}
	}
	// it stopped moving at minute 8 and the next trip began at minute 50: 42 minutes, not the 30 after the engine went off
	if got == nil || got.Start != t0.Add(8*time.Minute).Unix() || got.Minutes != 42 {
		t.Fatalf("the wait with the engine on counts: %+v in %s", got, body)
	}
}
