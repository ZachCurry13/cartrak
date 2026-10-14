package yearreport

import (
	"math"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

func trip(id int64, month time.Month, km float64, cat string, kwh float64) *store.Trip {
	ts := time.Date(2026, month, 10, 12, 0, 0, 0, time.UTC).Unix()
	s, e := 50.0, 40.0
	t := &store.Trip{ID: id, StartTS: ts, DistanceKm: km, Category: cat, StartKwh: &s, EndKwh: &e}
	*t.StartKwh = kwh + 10
	*t.EndKwh = 10
	return t
}

func TestBuildTotalsADriveOfTheYear(t *testing.T) {
	r := Build([]*store.Trip{
		trip(1, time.January, 100, "work", 20),
		trip(2, time.January, 50, "personal", 10),
		trip(3, time.March, 300, "personal", 60),
		trip(4, time.March, 0.5, "personal", 0.01), // too short to count for efficiency
	}, nil, time.UTC)
	if r.Trips != 4 || math.Abs(r.Km-450.5) > 1e-9 || r.WorkKm != 100 || math.Abs(r.PersonalKm-350.5) > 1e-9 {
		t.Fatalf("%+v", r)
	}
	if r.EnergyTrips != 3 || math.Abs(r.EnergyKwh-90) > 1e-9 || math.Abs(r.EnergyKm-450) > 1e-9 || r.Estimated {
		t.Fatalf("energy: %+v", r)
	}
	if r.Longest == nil || r.Longest.ID != 3 || r.BusiestKm != "2026-03" || len(r.Months) != 2 || r.Months[0].Month != "2026-01" || r.Months[0].Trips != 2 {
		t.Fatalf("%+v", r)
	}
	if got := Build(nil, nil, time.UTC); got.Trips != 0 || got.Longest != nil || len(got.Months) != 0 {
		t.Fatalf("empty year: %+v", got)
	}
}
