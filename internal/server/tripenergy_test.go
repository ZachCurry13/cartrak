package server

import (
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/store"
)

func finishedTrip(t *testing.T, srv *Server, start time.Time, km float64, mod func(*store.Trip)) *store.Trip {
	tr := &store.Trip{StartTS: start.Unix(), Category: "work", CategorySource: "default"}
	mod(tr)
	if err := srv.S.CreateTrip(tr); err != nil {
		t.Fatal(err)
	}
	end := start.Add(30 * time.Minute).Unix()
	tr.EndTS, tr.DistanceKm, tr.Open = &end, km, false
	if err := srv.S.UpdateTrip(tr); err != nil {
		t.Fatal(err)
	}
	return tr
}

func TestTripsReportEnergyFromTheCarOrAnEstimate(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	cap := 88.0
	srv.S.SaveState("t", &ford.State{TS: time.Now(), CapacityKwh: &cap, Raw: []byte(`{}`)})
	day := time.Now().Add(-48 * time.Hour)
	v := func(x float64) *float64 { return &x }

	real := finishedTrip(t, srv, day, 40, func(tr *store.Trip) { tr.StartKwh, tr.EndKwh = v(80), v(70) })
	est := finishedTrip(t, srv, day.Add(2*time.Hour), 20, func(tr *store.Trip) { tr.StartSoC, tr.EndSoC = v(70), v(60) })
	none := finishedTrip(t, srv, day.Add(4*time.Hour), 5, func(tr *store.Trip) {})

	_, body := do(t, admin, "GET", ts.URL+"/api/trips", "", nil)
	var list []struct {
		ID              int64    `json:"id"`
		EnergyKwh       *float64 `json:"energyKwh"`
		EnergyEstimated bool     `json:"energyEstimated"`
		DistanceKm      float64  `json:"distanceKm"`
	}
	if err := json.Unmarshal([]byte(body), &list); err != nil {
		t.Fatal(err)
	}
	by := map[int64]int{}
	for i, x := range list {
		by[x.ID] = i
	}
	if r := list[by[real.ID]]; r.EnergyKwh == nil || *r.EnergyKwh != 10 || r.EnergyEstimated || r.DistanceKm != 40 {
		t.Fatalf("a trip with the car's own readings: %+v", r)
	}
	if e := list[by[est.ID]]; e.EnergyKwh == nil || *e.EnergyKwh < 8.79 || *e.EnergyKwh > 8.81 || !e.EnergyEstimated {
		t.Fatalf("an older trip is estimated from battery percent and flagged: %+v", e)
	}
	if n := list[by[none.ID]]; n.EnergyKwh != nil {
		t.Fatalf("no figure when nothing was recorded: %+v", n)
	}

	_, detail := do(t, admin, "GET", ts.URL+"/api/trips/"+strconv.FormatInt(real.ID, 10), "", nil)
	var d struct {
		Trip struct {
			EnergyKwh *float64 `json:"energyKwh"`
			StartKwh  *float64 `json:"startKwh"`
		} `json:"trip"`
	}
	json.Unmarshal([]byte(detail), &d)
	if d.Trip.EnergyKwh == nil || *d.Trip.EnergyKwh != 10 || d.Trip.StartKwh == nil {
		t.Fatalf("trip detail: %s", detail)
	}

	_, sum := do(t, admin, "GET", ts.URL+"/api/summary", "", nil)
	var s struct {
		Work struct {
			Trips    int     `json:"trips"`
			Kwh      float64 `json:"kwh"`
			EnergyKm float64 `json:"energyKm"`
		} `json:"work"`
	}
	json.Unmarshal([]byte(sum), &s)
	if s.Work.Trips != 3 || s.Work.Kwh < 18.79 || s.Work.Kwh > 18.81 || s.Work.EnergyKm != 60 {
		t.Fatalf("the period total counts only trips that have an energy figure (40 km + 20 km): %s", sum)
	}
}
