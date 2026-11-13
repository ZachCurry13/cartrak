package trips

import (
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/store"
)

// The pattern a hybrid sent: the ignition on with speed 0 for minutes, the odometer standing still, then 14.5 km further on
// still at speed 0, then a drive with a speed. Both parts are one drive from when the ignition went on.
func TestADriveSeenOnlyOnTheOdometerWithTheIgnitionOnIsRecorded(t *testing.T) {
	s, _ := store.Open(t.TempDir())
	t.Cleanup(func() { s.Close() })
	d := &Detector{S: s, TZ: time.UTC}
	t0 := time.Date(2026, 11, 9, 12, 10, 0, 0, time.UTC)
	at := func(min float64, ign string, odo, kph float64) *ford.State {
		return &ford.State{TS: t0.Add(time.Duration(min * float64(time.Minute))), Ignition: ign, OdometerKm: &odo, SpeedKph: &kph}
	}
	steps := []*ford.State{at(0, "OFF", 80000, 0), at(10, "ON", 80000, 0), at(14, "ON", 80000, 0), at(18, "ON", 80014.5, 0),
		at(21, "ON", 80014.5, 46), at(33, "OFF", 80029, 0)}
	for _, st := range steps {
		if err := d.Process(st); err != nil {
			t.Fatal(err)
		}
	}
	l, _ := s.ListTrips(store.TripFilter{})
	if len(l) != 1 {
		t.Fatalf("one drive, got %d trips", len(l))
	}
	if l[0].StartTS != t0.Add(10*time.Minute).Unix() || l[0].DistanceKm != 29 || l[0].Open {
		t.Fatalf("it starts when the ignition went on and counts the whole odometer rise: %+v", l[0])
	}
}

func TestSittingInTheCarWithTheIgnitionOnIsStillNotATrip(t *testing.T) {
	s, _ := store.Open(t.TempDir())
	t.Cleanup(func() { s.Close() })
	d := &Detector{S: s, TZ: time.UTC}
	t0 := time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)
	for i, odo := range []float64{500, 500, 500.4, 500.4} {
		kph := 0.0
		if err := d.Process(&ford.State{TS: t0.Add(time.Duration(i) * time.Minute), Ignition: "ON", OdometerKm: &odo, SpeedKph: &kph}); err != nil {
			t.Fatal(err)
		}
	}
	if l, _ := s.ListTrips(store.TripFilter{}); len(l) != 0 {
		t.Fatalf("less than a kilometer with no speed is not a drive: %+v", l)
	}
}
