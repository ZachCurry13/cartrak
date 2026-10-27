package store

import (
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
)

func TestATripWithNoStartPlaceGetsWhereTheCarWasLastSeen(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	la, lo := 44.5, -93.5
	if _, err := s.SaveState("t", &ford.State{TS: time.Unix(1000, 0), Lat: &la, Lon: &lo, Raw: []byte(`{}`)}); err != nil {
		t.Fatal(err)
	}
	tr := &Trip{StartTS: 2000, Category: "work", CategorySource: "default"}
	if err := s.CreateTrip(tr); err != nil {
		t.Fatal(err)
	}
	if err := s.FillTripStarts(); err != nil {
		t.Fatal(err)
	}
	got, _ := s.GetTrip(tr.ID)
	if got.StartLat == nil || *got.StartLat != la || *got.StartLon != lo {
		t.Fatalf("the start place is filled in: %+v", got)
	}
}
