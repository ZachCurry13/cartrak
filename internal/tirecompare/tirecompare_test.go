package tirecompare

import (
	"math"
	"testing"
)

// trips builds n trips of km each at tempC using kwhPerKm, on winter tires or not.
func trips(n int, km, tempC, kwhPerKm float64, winter bool) []Trip {
	out := make([]Trip, n)
	for i := range out {
		out[i] = Trip{ID: int64(i), Km: km, Kwh: km * kwhPerKm, TempC: tempC, HasT: true, AvgKph: 50, Winter: winter}
	}
	return out
}

func TestColdWinterTripsAgainstWarmSummerTripsAreNotComparedBecauseThatIsTheSeason(t *testing.T) {
	// summer tires only in mild weather, winter tires only in the cold: the winter trips use far more, but that is the cold.
	all := append(trips(10, 20, 18, 0.15, false), trips(10, 20, -8, 0.24, true)...)
	r := Compare(all)
	if r.Verdict != "none" || r.Matched != 0 || r.Pct != 0 {
		t.Fatalf("no overlap, no number: %+v", r)
	}
}

func TestTheSameTemperatureOnBothTiresGivesTheTirePenaltyAndNotTheWeather(t *testing.T) {
	var all []Trip
	for _, c := range []float64{-2, 3, 8} { // three temperature cells, the cold ones cost more for both sets
		base := 0.16 + (10-c)*0.004
		all = append(all, trips(8, 20, c, base, false)...)
		all = append(all, trips(8, 20, c, base*1.08, true)...) // the winter tires cost 8% in every cell
	}
	r := Compare(all)
	if r.Verdict != "good" || r.Matched != 3 || math.Abs(r.Pct-8) > 0.01 {
		t.Fatalf("8%% more with winter tires: %+v", r)
	}
	if math.Abs(r.LowPct-8) > 0.01 || math.Abs(r.HighPct-8) > 0.01 {
		t.Fatalf("the spread is the same everywhere: %+v", r)
	}
}

func TestACellNeedsEnoughOnBothSidesAndLittleOverlapIsNotAnAnswer(t *testing.T) {
	all := append(trips(2, 10, 3, 0.2, false), trips(2, 10, 3, 0.22, true)...) // 20 km each: under the 30 km a cell needs
	if r := Compare(all); r.Matched != 0 || r.Verdict != "none" {
		t.Fatalf("%+v", r)
	}
	all = append(trips(3, 15, 3, 0.2, false), trips(3, 15, 3, 0.22, true)...) // 45 km each: one cell, 45 km: under the 150 km that is "fair"
	if r := Compare(all); r.Matched != 1 || r.Verdict != "none" {
		t.Fatalf("one small cell is not enough: %+v", r)
	}
	all = append(trips(5, 20, 3, 0.2, false), trips(5, 20, 3, 0.22, true)...) // 100 km: still under
	all = append(all, append(trips(4, 20, 8, 0.19, false), trips(4, 20, 8, 0.21, true)...)...)
	if r := Compare(all); r.Verdict != "fair" || r.Matched != 2 {
		t.Fatalf("two cells, 180 km: fair: %+v", r)
	}
}

func TestCityAndHighwayAreNotMixedAndWetTripsAndShortTripsAreLeftOut(t *testing.T) {
	city := trips(4, 20, 3, 0.2, false)
	hwy := trips(4, 20, 3, 0.2, true)
	for i := range hwy {
		hwy[i].AvgKph = 100 // winter trips are all highway, the other trips all city: no common cell
	}
	if r := Compare(append(city, hwy...)); r.Matched != 0 {
		t.Fatalf("a highway trip is not compared with a city one: %+v", r)
	}
	wet := trips(4, 20, 3, 0.4, true)
	for i := range wet {
		wet[i].Wet = true
	}
	short := trips(10, 1, 3, 0.2, false)
	r := Compare(append(append(trips(4, 20, 3, 0.2, false), wet...), short...))
	if r.Excluded != 4 || r.WinterKm != 0 || r.OtherKm != 80 {
		t.Fatalf("wet and short trips do not count: %+v", r)
	}
}
