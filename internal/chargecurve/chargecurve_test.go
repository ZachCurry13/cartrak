package chargecurve

import (
	"math"
	"testing"
)

func TestACurveFromAFastChargeHasPeakAverageAndTheTenToEightyTime(t *testing.T) {
	// 5% to 90% in 40 minutes, power tapering as the battery fills.
	var ps []Point
	for i := 0; i <= 40; i += 4 {
		soc := 5 + 85*float64(i)/40
		kw := 150 - soc*1.2
		ps = append(ps, Point{TS: int64(i) * 60, SoC: soc, Kw: kw})
	}
	start, end := 4.0, 76.0 // kWh left
	c := Build(ps, &start, &end)
	if c.PeakKw < 140 || c.PeakKw > 150 {
		t.Fatalf("peak: %v", c.PeakKw)
	}
	if c.AddedKwh == nil || math.Abs(*c.AddedKwh-72) > 1e-9 || c.AvgKw == nil || math.Abs(*c.AvgKw-108) > 1e-6 {
		t.Fatalf("72 kWh in 40 minutes is 108 kW: %+v %+v", c.AddedKwh, c.AvgKw)
	}
	if c.Minutes10to80 == nil || *c.Minutes10to80 < 25 || *c.Minutes10to80 > 32 {
		t.Fatalf("10 to 80: %v", c.Minutes10to80)
	}
	if c.At80Kw == nil || *c.At80Kw > 60 {
		t.Fatalf("slower near 80: %v", c.At80Kw)
	}
}

func TestACurveWithoutEnergyUsesTheMeanPowerAndASessionAboveTenGivesNoTenToEighty(t *testing.T) {
	ps := []Point{{0, 40, 10}, {600, 45, 10.5}, {1200, 50, 0}}
	c := Build(ps, nil, nil)
	if c.AvgKw == nil || math.Abs(*c.AvgKw-10.25) > 1e-9 || c.AddedKwh != nil {
		t.Fatalf("%+v", c)
	}
	if c.Minutes10to80 != nil {
		t.Fatal("this session never covered 10 to 80")
	}
	if e := Build(nil, nil, nil); len(e.Points) != 0 || e.AvgKw != nil {
		t.Fatalf("%+v", e)
	}
}
