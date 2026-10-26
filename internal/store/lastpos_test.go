package store

import (
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
)

func TestTripsWithoutAStartPositionGetTheLastParkedOneOnce(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	lat, lon := 45.5, -84.7
	t0 := time.Date(2026, 10, 22, 8, 0, 0, 0, time.UTC)
	if _, err := s.SaveState("ford", &ford.State{TS: t0, Lat: &lat, Lon: &lon}); err != nil {
		t.Fatal(err)
	}
	tr := &Trip{StartTS: t0.Add(time.Hour).Unix(), Category: "personal", CategorySource: "default"}
	if err := s.CreateTrip(tr); err != nil {
		t.Fatal(err)
	}
	s.Close()
	s, err = Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if got, _ := s.GetTrip(tr.ID); got.StartLat != nil { // the repair already ran for this database, so a later trip is left alone
		t.Fatalf("runs once: %+v", got.StartLat)
	}
	if la, lo := s.LastPositionBefore(tr.StartTS); la == nil || *la != lat || *lo != lon {
		t.Fatal("last position before the trip")
	}
}
