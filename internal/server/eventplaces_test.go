package server

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/geocode"
	"github.com/ZachCurry13/cartrak/internal/store"
)

func TestTheDayViewSaysWhereEachDriveAndStopWas(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	f := func(x float64) *float64 { return &x }
	loc := srv.defaultLoc()
	day := time.Now().In(loc).AddDate(0, 0, -2)
	start := time.Date(day.Year(), day.Month(), day.Day(), 9, 0, 0, 0, loc)
	_ = srv.S.SetPlace(geocode.Key(44.76, -85.62), "Traverse City")
	_ = srv.S.SetPlace(geocode.Key(44.91, -85.99), "Glen Arbor Township")
	out := finishedTrip(t, srv, start, 30, func(tr *store.Trip) {
		tr.StartLat, tr.StartLon, tr.EndLat, tr.EndLon = f(44.76), f(-85.62), f(44.91), f(-85.99)
		tr.StartKwh, tr.EndKwh = f(60), f(54)
	})
	loop := finishedTrip(t, srv, start.Add(3*time.Hour), 5, func(tr *store.Trip) {
		tr.StartLat, tr.StartLon, tr.EndLat, tr.EndLon = f(44.91), f(-85.99), f(44.91), f(-85.99)
	})
	c := &store.Charge{StartTS: start.Add(time.Hour).Unix(), Lat: f(44.91), Lon: f(-85.99), StartKwh: f(54), PeakKw: 7}
	if err := srv.S.CreateCharge(c); err != nil {
		t.Fatal(err)
	}
	end := c.StartTS + 3600
	c.EndTS, c.EndKwh, c.Open = &end, f(61), false
	if err := srv.S.UpdateCharge(c); err != nil {
		t.Fatal(err)
	}
	_, body := do(t, admin, "GET", ts.URL+"/api/day?date="+day.Format("2006-01-02"), "", nil)
	var v struct {
		Events []struct {
			Kind      string   `json:"kind"`
			ID        int64    `json:"id"`
			Place     string   `json:"place"`
			FromPlace string   `json:"fromPlace"`
			EnergyKwh *float64 `json:"energyKwh"`
		} `json:"events"`
	}
	if err := json.Unmarshal([]byte(body), &v); err != nil {
		t.Fatalf("%s", body)
	}
	seen := 0
	for _, e := range v.Events {
		switch {
		case e.Kind == "driving" && e.ID == out.ID:
			seen++
			if e.FromPlace != "Traverse City" || e.Place != "Glen Arbor Township" || e.EnergyKwh == nil || *e.EnergyKwh != 6 {
				t.Fatalf("a drive from one city to another: %+v", e)
			}
		case e.Kind == "driving" && e.ID == loop.ID:
			seen++
			if e.FromPlace != "" || e.Place != "Around Glen Arbor Township" {
				t.Fatalf("a drive within one city: %+v", e)
			}
		case e.Kind == "charging":
			seen++
			if e.Place != "Glen Arbor Township" {
				t.Fatalf("a charge says where it was: %+v", e)
			}
		}
	}
	if seen != 3 {
		t.Fatalf("both drives and the charge: %s", body)
	}
}
