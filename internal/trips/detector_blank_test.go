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

func TestATripWithNoStartPositionBeginsWhereTheCarWasLastParked(t *testing.T) {
	s, err := store.Open(t.TempDir())
	t.Cleanup(func() { s.Close() })
	if err != nil {
		t.Fatal(err)
	}
	d := &Detector{S: s, TZ: time.UTC}
	t0 := time.Date(2026, 10, 22, 8, 0, 0, 0, time.UTC)
	lat, lon := 45.5, -84.7
	parked := reading(t0, "OFF", 1000, 80)
	parked.Lat, parked.Lon = &lat, &lon
	if _, err := s.SaveState("ford", parked); err != nil {
		t.Fatal(err)
	}
	for _, st := range []*ford.State{parked, reading(t0.Add(time.Hour), "ON", 1000, 80), reading(t0.Add(70*time.Minute), "ON", 1010, 77), reading(t0.Add(80*time.Minute), "OFF", 1015, 75)} {
		if _, err := s.SaveState("ford", st); err != nil {
			t.Fatal(err)
		}
		if err := d.Process(st); err != nil {
			t.Fatal(err)
		}
	}
	trips, _ := s.ListTrips(store.TripFilter{})
	if len(trips) != 1 || trips[0].StartLat == nil || *trips[0].StartLat != lat || *trips[0].StartLon != lon {
		t.Fatalf("it began where it was parked: %+v", trips)
	}
}

func TestAnOldLowerOdometerOnTheLastReadingDoesNotZeroTheDriveAndAStaleEndStaysUnknownUntilTheCarSaysWhere(t *testing.T) {
	s, err := store.Open(t.TempDir())
	t.Cleanup(func() { s.Close() })
	if err != nil {
		t.Fatal(err)
	}
	d := &Detector{S: s, TZ: time.UTC}
	t0 := time.Date(2026, 10, 23, 18, 0, 0, 0, time.UTC)
	lat, lon := 45.5, -84.7
	at := func(min int, ign string, odo float64, la, lo *float64) *ford.State {
		st := reading(t0.Add(time.Duration(min)*time.Minute), ign, odo, 80)
		st.Lat, st.Lon = la, lo
		if ign == "ON" {
			sp := 80.0
			st.SpeedKph = &sp
		}
		return st
	}
	for _, st := range []*ford.State{at(0, "OFF", 1000, &lat, &lon), at(1, "ON", 1000, &lat, &lon), at(10, "ON", 1019, &lat, &lon), at(20, "OFF", 1000, &lat, &lon)} { // the last one reports the old odometer
		if err := d.Process(st); err != nil {
			t.Fatal(err)
		}
	}
	trips, _ := s.ListTrips(store.TripFilter{})
	if len(trips) != 1 || trips[0].DistanceKm != 19 || trips[0].EndLat != nil {
		t.Fatalf("19 km, the end unknown: %+v", trips)
	}
	nl, no := 45.9, -84.1
	if err := d.Process(at(25, "OFF", 1019, &nl, &no)); err != nil {
		t.Fatal(err)
	}
	got, _ := s.GetTrip(trips[0].ID)
	if got.EndLat == nil || *got.EndLat != nl {
		t.Fatalf("the first new position becomes the end: %+v", got.EndLat)
	}
}
