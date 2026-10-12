package trips

import (
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/store"
)

func reading(ts time.Time, ign string, odo, soc float64) *ford.State {
	return &ford.State{TS: ts, Ignition: ign, OdometerKm: &odo, SoCPct: &soc}
}

func TestTripLifecycleAndRule(t *testing.T) {
	s, err := store.Open(t.TempDir())
	t.Cleanup(func() { s.Close() })
	if err != nil {
		t.Fatal(err)
	}
	s.SaveRule(store.Rule{Name: "Work hours", Category: Work, Enabled: true, Days: 62, StartMin: 480, EndMin: 1020})
	d := &Detector{S: s, TZ: time.UTC}
	t0 := time.Date(2026, 10, 12, 8, 30, 0, 0, time.UTC) // Monday 8:30

	steps := []*ford.State{
		reading(t0, "OFF", 1000, 80),
		reading(t0.Add(1*time.Minute), "ON", 1000, 80),
		reading(t0.Add(10*time.Minute), "ON", 1012, 76),
		reading(t0.Add(20*time.Minute), "OFF", 1020, 74),
	}
	for _, st := range steps {
		if err := d.Process(st); err != nil {
			t.Fatal(err)
		}
	}
	trips, _ := s.ListTrips(store.TripFilter{})
	if len(trips) != 1 {
		t.Fatalf("want 1 trip, got %d", len(trips))
	}
	tr := trips[0]
	if tr.Open || tr.DistanceKm != 20 || tr.Category != Work || tr.CategorySource != "rule" {
		t.Fatalf("bad trip: %+v", tr)
	}

	// A manual choice survives re-running rules.
	s.SetTripCategory(tr.ID, Personal, "dentist")
	if err := d.Reclassify(); err != nil {
		t.Fatal(err)
	}
	tr, _ = s.GetTrip(tr.ID)
	if tr.Category != Personal || tr.CategorySource != "manual" {
		t.Fatalf("manual override lost: %+v", tr)
	}
}

func TestIgnitionBlipIsDiscarded(t *testing.T) {
	s, _ := store.Open(t.TempDir())
	t.Cleanup(func() { s.Close() })
	d := &Detector{S: s, TZ: time.UTC}
	t0 := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	d.Process(reading(t0, "ON", 50, 50))
	d.Process(reading(t0.Add(30*time.Second), "OFF", 50, 50))
	if ts, _ := s.ListTrips(store.TripFilter{}); len(ts) != 0 {
		t.Fatalf("blip kept: %d", len(ts))
	}
}

func TestChargeSession(t *testing.T) {
	s, _ := store.Open(t.TempDir())
	t.Cleanup(func() { s.Close() })
	d := &Detector{S: s, TZ: time.UTC}
	t0 := time.Now()
	kw := 11.0
	a := reading(t0, "OFF", 10, 40)
	a.ChargeState = "ChargingAC"
	a.ChargeKw = &kw
	b := reading(t0.Add(time.Hour), "OFF", 10, 70)
	b.ChargeState = "ChargeCompleted"
	d.Process(a)
	d.Process(b)
	cs, _ := s.ListCharges(10)
	if len(cs) != 1 || cs[0].Open || *cs[0].EndSoC != 70 || cs[0].PeakKw != 11 {
		t.Fatalf("bad charge: %+v", cs)
	}
}
