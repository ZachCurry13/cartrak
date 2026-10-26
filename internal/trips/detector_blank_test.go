package trips

import (
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/store"
)

func TestAnEmptyAnswerFromFordDoesNotEndADriveInProgress(t *testing.T) {
	s, err := store.Open(t.TempDir())
	t.Cleanup(func() { s.Close() })
	if err != nil {
		t.Fatal(err)
	}
	d := &Detector{S: s, TZ: time.UTC}
	t0 := time.Date(2026, 10, 22, 16, 0, 0, 0, time.UTC)
	speed := 100.0
	moving := func(at time.Duration, odo float64) *ford.State {
		st := reading(t0.Add(at), "ON", odo, 70)
		st.SpeedKph = &speed
		return st
	}
	for _, st := range []*ford.State{moving(0, 1000), moving(10*time.Minute, 1015), {TS: t0.Add(11 * time.Minute)}, moving(21*time.Minute, 1030), reading(t0.Add(30*time.Minute), "OFF", 1040, 66)} {
		if err := d.Process(st); err != nil {
			t.Fatal(err)
		}
	}
	trips, _ := s.ListTrips(store.TripFilter{})
	if len(trips) != 1 || trips[0].DistanceKm != 40 {
		t.Fatalf("one drive of 40 km, not two: %+v", trips)
	}
}
