package trips

import (
	"testing"

	"github.com/ZachCurry13/cartrak/internal/store"
)

func fl(x float64) *float64 { return &x }

func TestEnergyPrefersTheCarsOwnReading(t *testing.T) {
	e, ok := EnergyUsed(&store.Trip{StartKwh: fl(80), EndKwh: fl(72.5), StartSoC: fl(90), EndSoC: fl(82)}, fl(88))
	if !ok || e.Estimated || e.Kwh != 7.5 {
		t.Fatalf("the energy-left readings win: %+v %v", e, ok)
	}
}

func TestEnergyFallsBackToBatteryPercentAndSaysSo(t *testing.T) {
	e, ok := EnergyUsed(&store.Trip{StartSoC: fl(80), EndSoC: fl(70)}, fl(88))
	if !ok || !e.Estimated || e.Kwh < 8.79 || e.Kwh > 8.81 {
		t.Fatalf("10 percent of 88 kWh is 8.8: %+v %v", e, ok)
	}
	if _, ok := EnergyUsed(&store.Trip{StartSoC: fl(80), EndSoC: fl(70)}, nil); ok {
		t.Fatal("without a capacity there is nothing to estimate from")
	}
}

func TestNoFigureIsInventedWhenThereIsNoRealUse(t *testing.T) {
	for name, tr := range map[string]*store.Trip{
		"regenerated more than used": {StartKwh: fl(70), EndKwh: fl(71)},
		"identical readings":         {StartKwh: fl(70), EndKwh: fl(70)},
		"percent barely moved":       {StartSoC: fl(70), EndSoC: fl(69.8)},
		"nothing recorded":           {},
		"only a start reading":       {StartKwh: fl(70)},
	} {
		if e, ok := EnergyUsed(tr, fl(88)); ok {
			t.Errorf("%s: should have no figure, got %+v", name, e)
		}
	}
}
