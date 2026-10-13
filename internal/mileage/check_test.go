package mileage

import (
	"math"
	"testing"

	"github.com/ZachCurry13/cartrak/internal/store"
)

func odo(ts int64, km float64) store.OdoPoint { return store.OdoPoint{TS: ts, Km: km} }

func tripOdo(id int64, from, to float64) *store.Trip {
	return &store.Trip{ID: id, StartOdo: &from, EndOdo: &to}
}

func TestEveryKilometreRecordedMeansItAddsUp(t *testing.T) {
	pts := []store.OdoPoint{odo(0, 1000), odo(60, 1010), odo(120, 1030), odo(180, 1030), odo(240, 1045)}
	c := CheckTrips(pts, []*store.Trip{tripOdo(1, 1000, 1030), tripOdo(2, 1030, 1045)})
	if !c.Ok || c.DrivenKm != 45 || c.MissingKm != 0 || len(c.Gaps) != 0 || c.CoveredKm != 45 {
		t.Fatalf("%+v", c)
	}
}

func TestAMissedTripShowsUpAsAGapWithItsWindow(t *testing.T) {
	// 1000 to 1020 recorded, then 1020 to 1050 missed entirely (two readings while driving), then 1050 to 1060 recorded.
	pts := []store.OdoPoint{odo(0, 1000), odo(60, 1020), odo(3600, 1030), odo(3660, 1050), odo(7200, 1060)}
	c := CheckTrips(pts, []*store.Trip{tripOdo(1, 1000, 1020), tripOdo(2, 1050, 1060)})
	if c.Ok || len(c.Gaps) != 1 {
		t.Fatalf("%+v", c)
	}
	g := c.Gaps[0]
	if g.FromTS != 60 || g.ToTS != 3660 || g.StartOdo != 1020 || g.EndOdo != 1050 || math.Abs(g.Km-30) > 1e-9 || g.Edge != "" {
		t.Fatalf("a whole missed trip, not an edge: %+v", g)
	}
	if c.DrivenKm != 60 || c.MissingKm != 30 || c.CoveredKm != 30 {
		t.Fatalf("totals: %+v", c)
	}
}

func TestAStretchBeforeARecordedTripIsItsLateStart(t *testing.T) {
	// The trip was only noticed at 1003, so the first 3 km are missing from it.
	pts := []store.OdoPoint{odo(0, 1000), odo(300, 1003), odo(900, 1020)}
	c := CheckTrips(pts, []*store.Trip{tripOdo(7, 1003, 1020)})
	if len(c.Gaps) != 1 || c.Gaps[0].Edge != "start" || c.Gaps[0].TripID != 7 {
		t.Fatalf("%+v", c)
	}
	// And one noticed too late at the end.
	pts = []store.OdoPoint{odo(0, 1000), odo(300, 1015), odo(900, 1018)}
	c = CheckTrips(pts, []*store.Trip{tripOdo(8, 1000, 1015)})
	if len(c.Gaps) != 1 || c.Gaps[0].Edge != "end" || c.Gaps[0].TripID != 8 {
		t.Fatalf("%+v", c)
	}
}

func TestRoundingNoiseAndParkedReadingsAreNotGaps(t *testing.T) {
	pts := []store.OdoPoint{odo(0, 1000), odo(60, 1000.1), odo(120, 1000.2), odo(180, 1000.2)}
	if c := CheckTrips(pts, nil); !c.Ok || len(c.Gaps) != 0 {
		t.Fatalf("%+v", c)
	}
	if c := CheckTrips([]store.OdoPoint{odo(0, 5)}, nil); !c.Ok || c.Readings != 1 {
		t.Fatalf("one reading proves nothing: %+v", c)
	}
	// A gap of only a few hundred metres in total is not worth the owner's attention.
	if c := CheckTrips([]store.OdoPoint{odo(0, 100), odo(60, 100.6)}, nil); len(c.Gaps) != 0 {
		t.Fatalf("under a kilometre: %+v", c)
	}
}

func TestSeparateMissedDrivesAreSeparateGaps(t *testing.T) {
	pts := []store.OdoPoint{odo(0, 1000), odo(60, 1010), odo(4000, 1010), odo(4060, 1025)}
	c := CheckTrips(pts, nil)
	if len(c.Gaps) != 2 || c.Gaps[0].Km != 10 || c.Gaps[1].Km != 15 || c.MissingKm != 25 {
		t.Fatalf("a stop between them keeps them apart: %+v", c)
	}
}
