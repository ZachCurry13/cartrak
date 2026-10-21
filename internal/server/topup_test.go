package server

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

func TestAHomeTopUpBeforeADriveIsNotCountedAsACharge(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	f := func(x float64) *float64 { return &x }
	loc := srv.defaultLoc()
	now := time.Now().In(loc)
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, -2)
	if err := srv.S.SaveNamedPlace(&store.NamedPlace{Name: "Home", Lat: 45.5, Lon: -84.0, RadiusM: 150}); err != nil {
		t.Fatal(err)
	}
	mk := func(at time.Time, mins int64, from, to float64, lat, lon float64) {
		c := &store.Charge{StartTS: at.Unix(), Lat: f(lat), Lon: f(lon), StartKwh: f(from), PeakKw: 7}
		if err := srv.S.CreateCharge(c); err != nil {
			t.Fatal(err)
		}
		end := c.StartTS + mins*60
		c.EndTS, c.EndKwh, c.Open = &end, f(to), false
		if err := srv.S.UpdateCharge(c); err != nil {
			t.Fatal(err)
		}
	}
	mk(day.Add(3*time.Hour+30*time.Minute), 50, 88.0, 88.4, 45.5, -84.0) // the car finishing its charge at home
	finishedTrip(t, srv, day.Add(4*time.Hour+25*time.Minute), 160, func(tr *store.Trip) {
		tr.StartLat, tr.StartLon, tr.EndLat, tr.EndLon = f(45.5), f(-84.0), f(45.1), f(-87.6)
	})
	mk(day.Add(6*time.Hour), 28, 45, 71, 45.1, -87.6) // a real charge on the road
	_, body := do(t, admin, "GET", ts.URL+"/api/day?date="+day.Format("2006-01-02"), "", nil)
	var v struct {
		Summary struct {
			Sessions int `json:"sessions"`
		} `json:"summary"`
	}
	if err := json.Unmarshal([]byte(body), &v); err != nil || v.Summary.Sessions != 1 {
		t.Fatalf("only the charge on the road counts: %s", body)
	}
}
