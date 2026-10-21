package analytics

import (
	"math"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/fuel"
)

func fp2(v float64) *float64 { return &v }

func TestAMonthOfFuel(t *testing.T) {
	day := func(d string) int64 {
		return time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC).AddDate(0, 0, int(d[0]-'0')).Unix()
	}
	ts := []Trip{
		{ID: 1, Start: day("1"), Km: 80, Gal: 2},   // 40 km/gal
		{ID: 2, Start: day("3"), Km: 40, Gal: 1.2}, // 33 km/gal
		{ID: 3, Start: day("4"), Km: 20},           // no gauge reading: counts as driving, not as fuel
	}
	fills := []fuel.Fill{
		{TS: day("2"), Gallons: fp2(10), PricePerGal: fp2(3), OdoKm: fp2(1000)},
		{TS: day("5"), Gallons: fp2(5), PricePerGal: fp2(4), OdoKm: fp2(1140)},
		{TS: day("5"), StartPct: 10, EndPct: 50, Status: "pending"}, // nothing entered yet: left out
	}
	from, to, _, _ := Period("2026-10", time.UTC)
	r := FuelSummary(from, to, ts, fills)
	if math.Abs(r.Gal-3.2) > 1e-9 || r.KmFueled != 120 || math.Abs(r.KmPerGal-37.5) > 1e-9 || r.Fills != 2 || r.GalBought != 15 || r.Spent != 50 ||
		math.Abs(r.AvgPrice-50.0/15) > 1e-9 || math.Abs(r.CostPerKm-50.0/140) > 1e-9 {
		t.Fatalf("the month: %+v", r)
	}
	if m := FuelMonths(ts, fills, time.UTC); len(m) != 1 || m[0].Period != "2026-10" || m[0].Spent != 50 {
		t.Fatalf("months: %+v", m)
	}
	recs := map[string]Record{}
	for _, x := range FuelRecords(ts, fills) {
		recs[x.Key] = x
	}
	if recs["bestMpg"].TripID != 1 || recs["bigfill"].Value != 0 || recs["biggestFill"].Value != 10 || recs["cheapestGas"].Value != 3 || recs["longestTank"].Value != 140 {
		t.Fatalf("records: %+v", recs)
	}
}

func TestAGaugeThatBarelyMovedIsNotTheFuelRecord(t *testing.T) {
	var ts []Trip
	for i := int64(1); i <= 6; i++ { // a car that gets about 36 km/gal
		ts = append(ts, Trip{ID: i, Start: i * 86400, Km: 36, Gal: 1})
	}
	ts = append(ts, Trip{ID: 7, Start: 7 * 86400, Km: 40, Gal: 0.5}) // 80 km/gal: the gauge did not fall
	ts = append(ts, Trip{ID: 8, Start: 8 * 86400, Km: 40, Gal: 0.8}) // 50 km/gal: a good drive
	for _, r := range FuelRecords(ts, nil) {
		if r.Key == "bestMpg" && r.TripID != 8 {
			t.Fatalf("the best believable drive wins: %+v", r)
		}
	}
}
