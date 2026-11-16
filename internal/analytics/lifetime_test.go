package analytics

import (
	"math"
	"strings"
	"testing"
)

// A month of cold driving at 3 mi/kWh (4.83 km/kWh) on a car with 146,343 km on the clock, 140,000 of them before tracking.
func coldMonth() ([]Trip, []Charge) {
	cold := 2.0
	var ts []Trip
	odo := 140000.0
	for i := 0; i < 30; i++ {
		odo += 40
		ts = append(ts, Trip{ID: int64(i), Start: int64(1_760_000_000 + i*86400), Km: 40, Kwh: 40 / 4.83, OdoKm: odo, TempC: &cold})
	}
	cost := 0.18 * 300
	cs := []Charge{{Kwh: 270, WallKwh: 300, Cost: &cost, GasOK: true, GasCost: 81, GasLiters: 76}}
	return ts, cs
}

func TestTheLifetimeIsARangeFromColdToMildDriving(t *testing.T) {
	ts, cs := coldMonth()
	l, ok := EstimateLifetime(ts, cs, 146343)
	if !ok {
		t.Fatal("enough driving")
	}
	if math.Abs(l.BeforeKm-140000) > 1 || l.MildMeasured || math.Abs(l.MildEff-4.83*1.2) > 0.01 {
		t.Fatalf("before %v, mild %v", l.BeforeKm, l.MildEff)
	}
	if math.Abs(l.BatteryHigh-146343/4.83) > 5 || math.Abs(l.BatteryLow-146343/(4.83*1.2)) > 5 || l.BatteryKwh <= l.BatteryLow || l.BatteryKwh >= l.BatteryHigh {
		t.Fatalf("range %v to %v, middle %v", l.BatteryLow, l.BatteryHigh, l.BatteryKwh)
	}
	if math.Abs(l.WallKwh-l.BatteryKwh*300/270) > 1 || l.Cost == nil || math.Abs(*l.Cost-l.WallKwh*0.18) > 1 || l.GasCost == nil || math.Abs(*l.GasCost-l.BatteryKwh*81/270) > 1 {
		t.Fatalf("wall %v cost %v gas %v", l.WallKwh, l.Cost, l.GasCost)
	}
	if _, ok := EstimateLifetime(ts[:3], cs, 146343); ok {
		t.Fatal("three trips are too few to go on")
	}
}

func TestTrophiesReachedBeforeTrackingAreMarkedAndEnergyIsNeverEarnedTooEarly(t *testing.T) {
	ts, cs := coldMonth()
	l, _ := EstimateLifetime(ts, cs, 146343)
	in := []Badge{{Key: "km16000", Goal: 16093, Value: 1200}, {Key: "world", Goal: 40075, Value: 1200}, {Key: "mwh10", Goal: 10000, Value: 300},
		{Key: "mwh100", Goal: 100000, Value: 300}, {Key: "kw150", Goal: 150}}
	out := WithLifetime(in, l)
	for _, b := range out[:3] {
		if b.Earned != l.Since || !strings.Contains(b.Note, "CarTrak began") {
			t.Fatalf("%s reached before tracking: %+v", b.Key, b)
		}
	}
	m := out[3]
	if m.Earned != 0 || m.Value < 24000 || !strings.Contains(m.Note, "MWh estimated from the odometer") {
		t.Fatalf("100 MWh not reached, but counts the estimate: %+v", m)
	}
	if out[4].Earned != 0 || out[4].Note != "" {
		t.Fatalf("a moment trophy is left alone: %+v", out[4])
	}
}
