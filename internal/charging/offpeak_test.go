package charging

import (
	"math"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

func TestWindowShareAcrossMidnightAndWithinADay(t *testing.T) {
	loc := time.UTC
	at := func(h, m int) int64 { return time.Date(2026, 10, 13, h, m, 0, 0, loc).Unix() }
	night := Window{22 * 60, 6 * 60}
	if s := night.Share(at(23, 0), at(23, 0)+3600, loc); math.Abs(s-1) > 1e-9 {
		t.Fatalf("all inside: %v", s)
	}
	if s := night.Share(at(12, 0), at(14, 0), loc); s != 0 {
		t.Fatalf("all outside: %v", s)
	}
	// 20:00 to 00:00: half of it (22:00 to 24:00) is in the window.
	if s := night.Share(at(20, 0), at(20, 0)+4*3600, loc); math.Abs(s-0.5) > 0.01 {
		t.Fatalf("half: %v", s)
	}
	day := Window{9 * 60, 17 * 60}
	if s := day.Share(at(8, 0), at(10, 0), loc); math.Abs(s-0.5) > 0.01 {
		t.Fatalf("a window inside one day: %v", s)
	}
	if s := night.Share(at(23, 0), at(23, 0), loc); s != 1 {
		t.Fatalf("an instant: %v", s)
	}
}

func TestHomeChargingInTheCheapHoursIsPricedWithTheCheaperRate(t *testing.T) {
	loc := time.UTC
	start := time.Date(2026, 10, 13, 20, 0, 0, 0, loc).Unix()
	end := start + 4*3600 // 20:00 to 00:00, half of it in 22:00-06:00
	kwh0, kwh1 := 0.0, 18.0
	c := &store.Charge{StartTS: start, EndTS: &end, StartKwh: &kwh0, EndKwh: &kwh1, PeakKw: 7, Kind: "home"}
	hi, lo := 0.20, 0.10
	p := Prices{Home: &hi, OffPeak: &lo, Window: &Window{22 * 60, 6 * 60}, Loc: loc, AcEfficiency: 0.9}
	s := Enrich(c, nil, nil, p) // 18 kWh added, 20 kWh from the wall; blended price 0.15
	if s.Cost == nil || math.Abs(*s.Cost-3.0) > 0.05 {
		t.Fatalf("half at 0.20 and half at 0.10 on 20 kWh is 3.00: %+v", s.Cost)
	}
	p.OffPeak = nil
	if s := Enrich(c, nil, nil, p); s.Cost == nil || math.Abs(*s.Cost-4.0) > 1e-9 {
		t.Fatalf("no cheap plan: %+v", s.Cost)
	}
}

func TestForecastShowsWhatTheCheapHoursWouldCost(t *testing.T) {
	hi, lo := 0.20, 0.08
	fc, ok := ForecastCharge(90, 20, 80, Prices{Home: &hi, OffPeak: &lo}, 0.9)
	if !ok || fc.OffPeakCost == nil || math.Abs(*fc.OffPeakCost-60*0.08) > 1e-9 || math.Abs(*fc.HomeCost-12) > 1e-9 {
		t.Fatalf("%+v", fc)
	}
}
