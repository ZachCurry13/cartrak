package charging

import (
	"math"
	"testing"
)

func TestForecastChargePricesHomeWithLossAndPublicWithout(t *testing.T) {
	home, pub := 0.10, 0.40
	f, ok := ForecastCharge(90, 20, 80, Prices{Home: &home, Public: &pub}, 0.9)
	if !ok || math.Abs(f.BatteryKwh-54) > 1e-9 {
		t.Fatalf("%+v %v", f, ok)
	}
	if math.Abs(f.HomeKwh-60) > 1e-9 || math.Abs(*f.HomeCost-6) > 1e-9 {
		t.Fatalf("home: %+v", f)
	}
	if math.Abs(*f.PublicCost-21.6) > 1e-9 {
		t.Fatalf("public: %+v", f)
	}
}

func TestForecastChargeRejectsNonsenseAndMissingPrices(t *testing.T) {
	for _, c := range [][3]float64{{0, 20, 80}, {90, 80, 20}, {90, 50, 50}, {90, -1, 50}, {90, 20, 101}} {
		if _, ok := ForecastCharge(c[0], c[1], c[2], Prices{}, 0.9); ok {
			t.Fatalf("accepted %v", c)
		}
	}
	f, ok := ForecastCharge(90, 20, 80, Prices{}, 0.9)
	if !ok || f.HomeCost != nil || f.PublicCost != nil {
		t.Fatalf("no prices means no cost: %+v", f)
	}
}

func TestMonthProjection(t *testing.T) {
	if p, ok := MonthProjection(30, 10, 30); !ok || math.Abs(p-90) > 1e-9 {
		t.Fatalf("%v %v", p, ok)
	}
	if _, ok := MonthProjection(30, 2, 30); ok {
		t.Fatal("too early to project")
	}
	if p, ok := MonthProjection(30, 30, 30); !ok || p != 30 {
		t.Fatalf("last day: %v", p)
	}
	if _, ok := MonthProjection(0, 15, 30); ok {
		t.Fatal("nothing spent")
	}
}
