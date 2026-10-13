package server

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/store"
)

type checkOut struct {
	Check struct {
		DrivenKm  float64 `json:"drivenKm"`
		MissingKm float64 `json:"missingKm"`
		Ok        bool    `json:"ok"`
		Gaps      []struct {
			FromTS   int64    `json:"fromTs"`
			ToTS     int64    `json:"toTs"`
			StartOdo float64  `json:"startOdo"`
			EndOdo   float64  `json:"endOdo"`
			Km       float64  `json:"km"`
			Edge     string   `json:"edge"`
			TripID   int64    `json:"tripId"`
			StartLat *float64 `json:"startLat"`
			EndLat   *float64 `json:"endLat"`
		} `json:"gaps"`
	} `json:"check"`
}

func TestTheMileageCheckFindsAMissedTripAndOneTapFixesIt(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	day := time.Date(2026, 9, 15, 8, 0, 0, 0, time.UTC)
	lon := -75.0
	reading := func(at time.Time, odo float64, la float64) {
		st := &ford.State{TS: at, OdometerKm: &odo, Lat: &la, Lon: &lon, Raw: []byte(`{}`)}
		if _, err := srv.S.SaveState("t", st); err != nil {
			t.Fatal(err)
		}
	}
	// Parked at 1000, a drive to 1020 that CarTrak recorded, a drive to 1050 that it missed, then parked.
	reading(day, 1000, 40.00)
	reading(day.Add(10*time.Minute), 1020, 40.10)
	reading(day.Add(2*time.Hour), 1020, 40.10)
	reading(day.Add(2*time.Hour+20*time.Minute), 1050, 40.30)
	recorded := &store.Trip{StartTS: day.Unix(), EndTS: new(int64), StartOdo: new(float64), EndOdo: new(float64), DistanceKm: 20, Category: "work", TZ: "UTC"}
	*recorded.EndTS, *recorded.StartOdo, *recorded.EndOdo = day.Add(10*time.Minute).Unix(), 1000, 1020
	if err := srv.S.AddManualTrip(recorded); err != nil {
		t.Fatal(err)
	}

	url := ts.URL + "/api/mileage/check?from=2026-09-15&to=2026-09-15"
	get := func() checkOut {
		code, b := do(t, admin, "GET", url, "", nil)
		var o checkOut
		if code != 200 || json.Unmarshal([]byte(b), &o) != nil {
			t.Fatalf("check: %d %s", code, b)
		}
		return o
	}
	o := get()
	if o.Check.Ok || o.Check.DrivenKm != 50 || math.Abs(o.Check.MissingKm-30) > 1e-6 || len(o.Check.Gaps) != 1 {
		t.Fatalf("30 of 50 km were missed: %+v", o.Check)
	}
	g := o.Check.Gaps[0]
	if g.StartOdo != 1020 || g.EndOdo != 1050 || g.Edge != "" || g.StartLat == nil || g.EndLat == nil || *g.EndLat != 40.30 {
		t.Fatalf("the gap and where the car was at each end: %+v", g)
	}

	add := fmt.Sprintf(`{"startTs":%d,"endTs":%d,"startOdo":%v,"endOdo":%v,"startLat":%v,"startLon":%v,"endLat":%v,"endLon":%v,"category":"work"}`,
		g.FromTS, g.ToTS, g.StartOdo, g.EndOdo, *g.StartLat, lon, *g.EndLat, lon)
	if code, b := do(t, admin, "POST", ts.URL+"/api/trips/missed", add, xh); code != 200 {
		t.Fatalf("add: %d %s", code, b)
	}
	if o := get(); !o.Check.Ok || len(o.Check.Gaps) != 0 {
		t.Fatalf("after adding the trip it adds up: %+v", o.Check)
	}
	if code, _ := do(t, admin, "POST", ts.URL+"/api/trips/missed", add, xh); code != http.StatusConflict {
		t.Fatalf("adding the same miles twice is refused: %d", code)
	}
	if code, _ := do(t, admin, "POST", ts.URL+"/api/trips/missed", `{"startTs":1,"endTs":2,"startOdo":5,"endOdo":9,"category":"holiday"}`, xh); code != 400 {
		t.Fatalf("a made-up category: %d", code)
	}
	if code, _ := do(t, &http.Client{}, "GET", url, "", nil); code != 401 {
		t.Fatalf("anonymous: %d", code)
	}
	trips, _ := srv.S.ListTrips(store.TripFilter{Limit: 10})
	var added *store.Trip
	for _, x := range trips {
		if x.Notes == "Added from the mileage check" && x.StartOdo != nil && *x.StartOdo == 1020 {
			added = x
		}
	}
	if added == nil || added.Category != "work" || added.CategorySource != "manual" || added.DistanceKm != 30 || added.Open {
		t.Fatalf("the added trip: %+v", added)
	}

	// Extending a trip that was noticed late.
	late := fmt.Sprintf(`{"edge":"start","odo":%v}`, 1015.0)
	if code, b := do(t, admin, "POST", fmt.Sprintf("%s/api/trips/%d/extend", ts.URL, added.ID), late, xh); code != 200 {
		t.Fatalf("extend: %d %s", code, b)
	}
	if got, _ := srv.S.GetTrip(added.ID); got.DistanceKm != 35 || *got.StartOdo != 1015 {
		t.Fatalf("extended: %+v", got)
	}
	if code, _ := do(t, admin, "POST", fmt.Sprintf("%s/api/trips/%d/extend", ts.URL, added.ID), `{"edge":"start","odo":2000}`, xh); code != 400 {
		t.Fatalf("it can only grow outwards: %d", code)
	}
}
