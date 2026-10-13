package trips

import (
	"math"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

func driveAt(ts time.Time, km, from, to float64) *store.Trip {
	end := ts.Add(30 * time.Minute).Unix()
	return &store.Trip{StartTS: ts.Unix(), EndTS: &end, DistanceKm: km, StartSoC: &from, EndSoC: &to, TZ: "America/New_York"}
}

func TestRealDrivingGivesTheDistancePerPercentAndTheBand(t *testing.T) {
	day := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	ts := []*store.Trip{driveAt(day, 90, 80, 60), driveAt(day.Add(24*time.Hour), 45, 60, 50), driveAt(day.Add(48*time.Hour), 135, 50, 20)}
	w := Measure(ts)
	if w.Trips != 3 || w.Km != 270 || w.DropPct != 60 {
		t.Fatalf("window: %+v", w)
	}
	per, ok := w.KmPerPct()
	if !ok || math.Abs(per-4.5) > 1e-9 {
		t.Fatalf("4.5 km for every percent: %v %v", per, ok)
	}
	if got := KmBetween(per, 20, 80); math.Abs(got-270) > 1e-9 {
		t.Fatalf("20 to 80 percent is 60 percent of the battery: %v", got)
	}
	if KmBetween(per, 80, 20) != 0 || KmBetween(0, 20, 80) != 0 {
		t.Fatal("a backwards or unknown band is zero")
	}
}

func TestNoiseAndUnfinishedTripsAreLeftOut(t *testing.T) {
	day := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	short := driveAt(day, 1, 80, 78)   // too short to trust
	tiny := driveAt(day, 30, 80, 79.7) // under a percent
	gain := driveAt(day, 30, 40, 60)   // gained charge (downhill, or a bad reading)
	open := driveAt(day, 30, 80, 70)   // still in progress
	open.Open = true
	noSoC := &store.Trip{StartTS: day.Unix(), DistanceKm: 30}
	w := Measure([]*store.Trip{short, tiny, gain, open, noSoC})
	if w.Trips != 0 {
		t.Fatalf("none of these should count: %+v", w)
	}
	if _, ok := Measure([]*store.Trip{driveAt(day, 30, 80, 75)}).KmPerPct(); ok {
		t.Fatal("one short trip is not enough to say how far the car goes")
	}
}

func TestMonthsAreGroupedInTheTripsOwnTimeZoneNewestFirst(t *testing.T) {
	ny, _ := time.LoadLocation("America/New_York")
	// 01:00 on 1 July New York time is still June in UTC.
	july := driveAt(time.Date(2026, 7, 1, 1, 0, 0, 0, ny), 40, 80, 70)
	june := driveAt(time.Date(2026, 6, 15, 9, 0, 0, 0, ny), 50, 80, 70)
	got := ByMonth([]*store.Trip{june, july})
	if len(got) != 2 || got[0].Month != "2026-07" || got[1].Month != "2026-06" || got[0].Trips != 1 {
		t.Fatalf("months: %+v", got)
	}
}
