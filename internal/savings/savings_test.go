package savings

import (
	"math"
	"testing"
)

func near(a, b float64) bool { return math.Abs(a-b) < 0.01 }

func TestSavingsAgainstAGasCar(t *testing.T) {
	// 1000 km at 8 L/100 km and 1.50 a liter is 120 of gas; charging cost 40 and took 160 kWh from a 0.37 kg grid
	r := Compute(Input{Km: 1000, Sessions: 5, Priced: 5, ChargingCost: 40, WallKwh: 160, LitersPerKm: 0.08, PricePerLiter: 1.5, GridKgPerKwh: 0.37})
	if r.CostPerKm == nil || !near(*r.CostPerKm, 0.04) {
		t.Fatalf("cost per km: %+v", r.CostPerKm)
	}
	if r.GasCost == nil || !near(*r.GasCost, 120) || r.Saved == nil || !near(*r.Saved, 80) {
		t.Fatalf("gas cost and saving: %+v %+v", r.GasCost, r.Saved)
	}
	if r.CO2AvoidedKg == nil || !near(*r.CO2AvoidedKg, 80*2.348-160*0.37) {
		t.Fatalf("co2: %+v", r.CO2AvoidedKg)
	}
	if r.Partial {
		t.Fatal("every session is priced")
	}
}

func TestNothingIsComparedWithoutAGasCarAndCostPerKmNeedsEverySessionPriced(t *testing.T) {
	r := Compute(Input{Km: 500, Sessions: 4, Priced: 3, ChargingCost: 30})
	if r.Saved != nil || r.GasCost != nil || r.CO2AvoidedKg != nil {
		t.Fatalf("no gas car, no comparison: %+v", r)
	}
	if r.CostPerKm != nil || !r.Partial {
		t.Fatalf("one session has no price, so the cost per km is not stated: %+v", r)
	}
	if none := Compute(Input{Km: 0, Sessions: 1, Priced: 1, ChargingCost: 5, LitersPerKm: 0.08, PricePerLiter: 1.5}); none.Saved != nil || none.CostPerKm != nil {
		t.Fatalf("no distance, no figures: %+v", none)
	}
}

func TestTheSavingNeedsAtLeastOnePricedSession(t *testing.T) {
	r := Compute(Input{Km: 100, Sessions: 2, Priced: 0, LitersPerKm: 0.08, PricePerLiter: 1.5, WallKwh: 20})
	if r.Saved != nil || r.GasCost == nil || r.CO2AvoidedKg == nil {
		t.Fatalf("gas cost and CO2 are known, the saving is not: %+v", r)
	}
}
