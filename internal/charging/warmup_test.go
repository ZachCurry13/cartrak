package charging

import (
	"math"
	"testing"

	"github.com/ZachCurry13/cartrak/internal/store"
)

// A few minutes plugged in at home before leaving on a cold morning: the battery went from 75.9 to 75.5 kWh, but the car drew
// 0.7 kWh from the charger to warm itself. That is priced, and said to be a warm-up, not a charge.
func TestAWarmUpFromTheWallIsPricedByThePowerDrawn(t *testing.T) {
	f := func(v float64) *float64 { return &v }
	end := int64(500)
	c := &store.Charge{StartTS: 0, EndTS: &end, StartKwh: f(75.9), EndKwh: f(75.5), PeakKw: 6.4, Kind: "home", PowerKwh: f(0.7)}
	s := Enrich(c, f(88), nil, Prices{Home: f(0.15), AcEfficiency: 0.9})
	if !s.Warmup || s.AddedKwh != nil || s.Cost == nil || math.Abs(*s.Cost-0.7/0.9*0.15) > 1e-9 || s.PriceFrom != "home" {
		t.Fatalf("%+v cost %v", s, s.Cost)
	}
	none := &store.Charge{StartTS: 0, EndTS: &end, StartKwh: f(75.9), EndKwh: f(75.9), PeakKw: 6.4, Kind: "home"}
	if s := Enrich(none, f(88), nil, Prices{Home: f(0.15), AcEfficiency: 0.9}); s.Warmup || s.Cost != nil {
		t.Fatalf("no power reported, nothing to price: %+v", s)
	}
	fast := &store.Charge{StartTS: 0, EndTS: &end, StartKwh: f(40), EndKwh: f(40), PeakKw: 150, Kind: "public", PowerKwh: f(0.5)}
	if s := Enrich(fast, f(88), nil, Prices{Public: f(0.4), AcEfficiency: 0.9}); s.Warmup {
		t.Fatal("a fast charger that added nothing is not a warm-up at home")
	}
}
