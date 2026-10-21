package fuel

import (
	"math"
	"testing"
)

func f(v float64) *float64 { return &v }

func TestTankSizeIsLearnedFromFillUps(t *testing.T) {
	if g, n := TankGallons(nil); g != 0 || n != 0 {
		t.Fatal("nothing logged yet")
	}
	fills := []Fill{
		{StartPct: 20, EndPct: 100, Gallons: f(11.3), Status: "logged"}, // 14.1 gal
		{StartPct: 40, EndPct: 100, Gallons: f(8.6), Status: "logged"},  // 14.3 gal
		{StartPct: 10, EndPct: 100, Gallons: f(12.8), Status: "logged"}, // 14.2 gal
		{StartPct: 90, EndPct: 100, Gallons: f(1.4), Status: "logged"},  // a top-up: too coarse to say
		{StartPct: 20, EndPct: 100, Gallons: f(40), Status: "dismissed"},
		{StartPct: 20, EndPct: 100, Status: "pending"},
	}
	g, n := TankGallons(fills)
	if n != 3 || math.Abs(g-14.2) > 0.05 {
		t.Fatalf("tank: %.2f from %d", g, n)
	}
}

func TestFuelUsedAndMilesPerTank(t *testing.T) {
	if g, ok := Used(14.2, 80, 60); !ok || math.Abs(g-2.84) > 1e-9 {
		t.Fatalf("a fifth of 14.2 gallons: %v %v", g, ok)
	}
	if _, ok := Used(0, 80, 60); ok {
		t.Fatal("no tank size, no gallons")
	}
	if _, ok := Used(14.2, 80, 79.5); ok {
		t.Fatal("slosh is not fuel used")
	}
	if k, ok := KmPerTank(80, 80, 60); !ok || k != 400 {
		t.Fatalf("20%% of the tank in 80 km: a tank is 400 km: %v", k)
	}
}

func TestEconomyBetweenFillUps(t *testing.T) {
	prev := Fill{EndPct: 100, OdoKm: f(1000)}
	next := Fill{StartPct: 50, OdoKm: f(1300)}
	km, gal, kmpg := Between(prev, next, 14)
	if km != 300 || math.Abs(gal-7) > 1e-9 || math.Abs(kmpg-300.0/7) > 1e-9 {
		t.Fatalf("300 km on 7 gallons: %v %v %v", km, gal, kmpg)
	}
	if _, _, v := Between(prev, Fill{StartPct: 50}, 14); v != 0 {
		t.Fatal("no odometer, no economy")
	}
}

func TestRiseNeedsToBeFillingUp(t *testing.T) {
	if !Filled(f(20), f(75)) || Filled(f(50), f(55)) || Filled(f(70), f(40)) || Filled(nil, f(90)) {
		t.Fatal("only a real climb is a fill-up")
	}
}
