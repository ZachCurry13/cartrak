package server

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/store"
)

func TestTheWhyOfATripComparesItWithTheDriversOwnHistory(t *testing.T) {
	whyMu.Lock()
	whyAt = time.Time{}
	whyMu.Unlock()
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	fp := func(v float64) *float64 { return &v }
	t0 := time.Now().Add(-5 * time.Hour).Truncate(time.Second)
	odo, kwh := 9000.0, 85.0
	drive := func(start time.Time, n int, kph, perKm float64) {
		for i := 0; i <= n; i++ {
			st := &ford.State{TS: start.Add(time.Duration(i*30) * time.Second), OdometerKm: fp(odo), EnergyKwh: fp(kwh), SpeedKph: fp(kph), OutsideC: fp(15), Ignition: "ON", Raw: []byte(`{}`)}
			if _, err := srv.S.SaveState("t", st); err != nil {
				t.Fatal(err)
			}
			d := kph / 3600 * 30
			odo += d
			kwh -= d * perKm
		}
	}
	drive(t0, 120, 70, 0.15)                  // history: gentle
	drive(t0.Add(2*time.Hour), 120, 70, 0.15) // history: gentle
	start := t0.Add(3 * time.Hour)
	drive(start, 120, 70, 0.21) // the trip: same speed but much more energy per km
	end := start.Add(60 * time.Minute).Unix()
	km := 70.0
	tr := &store.Trip{StartTS: start.Unix(), EndTS: &end, DistanceKm: km, StartKwh: fp(85), EndKwh: fp(85 - km*0.21), Category: "personal", CategorySource: "default"}
	if err := srv.S.CreateTrip(tr); err != nil {
		t.Fatal(err)
	}
	tr.EndTS, tr.Open, tr.EndKwh = &end, false, fp(85-km*0.21)
	srv.S.UpdateTrip(tr)
	code, b := do(t, admin, "GET", fmt.Sprintf("%s/api/trips/%d/why", ts.URL, tr.ID), "", nil)
	var out struct {
		Ready      bool
		VsUsualPct float64
		Factors    []struct {
			Key string
			Pct float64
		}
	}
	if code != 200 || json.Unmarshal([]byte(b), &out) != nil || !out.Ready {
		t.Fatalf("%d %s", code, b)
	}
	if out.VsUsualPct < 10 {
		t.Fatalf("it used more than usual: %s", b)
	}
	if code, _ := do(t, admin, "GET", ts.URL+"/api/trips/9999/why", "", nil); code != 404 {
		t.Fatalf("no such trip: %d", code)
	}
	// Across the recent trips: how much energy each reason cost or saved against the usual.
	reasonsMu.Lock()
	reasonsOut = nil
	reasonsMu.Unlock()
	code, b = do(t, admin, "GET", ts.URL+"/api/efficiency-reasons?days=30", "", nil)
	var rr struct {
		Trips int
		Items []struct {
			Key string
			Kwh float64
		}
	}
	if code != 200 || json.Unmarshal([]byte(b), &rr) != nil || rr.Trips != 1 || len(rr.Items) != 3 {
		t.Fatalf("one trip, three reasons: %d %s", code, b)
	}
}
