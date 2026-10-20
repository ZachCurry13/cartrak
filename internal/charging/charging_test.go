package charging

import (
	"math"
	"testing"

	"github.com/ZachCurry13/cartrak/internal/store"
)

func f(v float64) *float64 { return &v }

func close(a *float64, want float64) bool { return a != nil && math.Abs(*a-want) < 1e-6 }

func TestEnergyAddedPrefersTheCarsOwnReadingAndFallsBackToPercent(t *testing.T) {
	own := Enrich(&store.Charge{StartKwh: f(40), EndKwh: f(58), PeakKw: 11}, nil, nil, Prices{})
	if !close(own.AddedKwh, 18) || own.Estimated || !close(own.WallKwh, 20) { // 18 / 0.9 on AC
		t.Fatalf("%+v", own)
	}
	est := Enrich(&store.Charge{StartSoC: f(20), EndSoC: f(70), PeakKw: 150}, f(80), nil, Prices{})
	if !close(est.AddedKwh, 40) || !est.Estimated || !close(est.WallKwh, 40) { // DC: no AC loss
		t.Fatalf("%+v", est)
	}
	if none := Enrich(&store.Charge{StartKwh: f(40), EndKwh: f(40)}, nil, nil, Prices{}); none.AddedKwh != nil {
		t.Fatal("nothing was added")
	}
}

func TestPlaceAndPriceDecideTheCost(t *testing.T) {
	home := &Point{Lat: 40.0, Lon: -75.0}
	prices := Prices{Home: f(0.10), Public: f(0.50)}
	at := func(lat, lon float64, kind string) Session {
		return Enrich(&store.Charge{StartKwh: f(10), EndKwh: f(28), PeakKw: 7, Lat: &lat, Lon: &lon, Kind: kind}, nil, home, prices)
	}
	h := at(40.0005, -75.0, "")                   // about 55 m away
	if h.Where != "home" || !close(h.Cost, 2.0) { // 20 kWh from the wall at 0.10
		t.Fatalf("home: %+v", h)
	}
	p := at(40.1, -75.0, "")
	if p.Where != "public" || !close(p.Cost, 10.0) {
		t.Fatalf("public: %+v", p)
	}
	if o := at(40.1, -75.0, "home"); o.Where != "home" || !close(o.Cost, 2.0) {
		t.Fatalf("the owner's choice wins: %+v", o)
	}
	if n := Enrich(&store.Charge{StartKwh: f(10), EndKwh: f(28), PeakKw: 7}, nil, nil, Prices{}); n.Cost != nil || n.Where != "" {
		t.Fatalf("no prices, no cost: %+v", n)
	}
}

func TestMonthsAddUpSessionsEnergyAndCost(t *testing.T) {
	mk := func(ts int64, kwh float64, cost *float64) Session {
		a := kwh
		return Session{Charge: &store.Charge{StartTS: ts}, AddedKwh: &a, Cost: cost}
	}
	month := func(ts int64) string {
		if ts < 100 {
			return "2026-08"
		}
		return "2026-09"
	}
	got := ByMonth([]Session{mk(10, 10, f(2)), mk(200, 5, f(1)), mk(300, 5, nil)}, month)
	if len(got) != 2 || got[0].Month != "2026-09" || got[0].Sessions != 2 || got[0].Kwh != 10 || got[0].Cost != 1 || got[0].Priced != 1 || got[1].Cost != 2 {
		t.Fatalf("%+v", got)
	}
}

func TestAnEnteredChargerNumberGivesTheRealEfficiencyAndIsLearned(t *testing.T) {
	mk := func(added, charger float64) Session {
		c := &store.Charge{StartKwh: f(10), EndKwh: f(10 + added), PeakKw: 7}
		if charger > 0 {
			c.ChargerKwh = f(charger)
		}
		return Enrich(c, nil, nil, Prices{Home: f(0.2), AcEfficiency: 0.9})
	}
	s := mk(18, 20.5)
	if !s.Measured || s.Efficiency == nil || math.Abs(*s.Efficiency-18/20.5) > 1e-9 || !close(s.WallKwh, 20.5) || !close(s.Cost, 4.1) {
		t.Fatalf("%+v", s)
	}
	if un := mk(18, 0); un.Measured || !close(un.WallKwh, 20) { // the 90% setting
		t.Fatalf("unmeasured uses the setting: %+v", un)
	}
	if low := mk(18, 10); low.Measured { // less than the battery got: a typing mistake, ignored
		t.Fatalf("an impossible number is not used: %+v", low)
	}
	eff, n := Learned([]Session{mk(18, 20.5), mk(30, 34), mk(10, 0)})
	if n != 2 || math.Abs(eff-48.0/54.5) > 1e-9 {
		t.Fatalf("energy-weighted average of the measured ones: %v %d", eff, n)
	}
	if _, n := Learned([]Session{mk(10, 0)}); n != 0 {
		t.Fatal("nothing measured")
	}
}

func TestWhatYouPaidReplacesTheCostWorkedOutFromAPrice(t *testing.T) {
	f := func(v float64) *float64 { return &v }
	price := 0.30
	c := &store.Charge{StartTS: 1000, EndTS: func() *int64 { v := int64(2000); return &v }(), StartKwh: f(10), EndKwh: f(50), PeakKw: 120, Kind: "public"}
	got := Enrich(c, f(88), nil, Prices{Public: &price})
	if got.CostActual || got.Cost == nil {
		t.Fatalf("a worked-out cost first: %+v", got)
	}
	c.ActualCost = f(13.06)
	got = Enrich(c, f(88), nil, Prices{Public: &price})
	if !got.CostActual || got.Cost == nil || *got.Cost != 13.06 {
		t.Fatalf("the receipt wins: %+v", got.Cost)
	}
}

func TestSlowChargingWallEnergyCountsWhatTheCarDrewBeyondTheBatteryRise(t *testing.T) {
	// the battery gained 0.4 kWh, but the car drew 2.0 kWh at the battery side while plugged in (warming up before leaving)
	topUp := Enrich(&store.Charge{StartKwh: f(83.8), EndKwh: f(84.2), PeakKw: 9.3, PowerKwh: f(2.0)}, nil, nil, Prices{Home: f(0.15)})
	if !close(topUp.WallKwh, 2.0/0.9) {
		t.Fatalf("wall energy should follow the reported power: %+v", topUp.WallKwh)
	}
	// when the battery rise is the larger figure, it still wins
	normal := Enrich(&store.Charge{StartKwh: f(40), EndKwh: f(58), PeakKw: 11, PowerKwh: f(17)}, nil, nil, Prices{})
	if !close(normal.WallKwh, 20) {
		t.Fatalf("%+v", normal.WallKwh)
	}
	// fast charging is untouched
	fast := Enrich(&store.Charge{StartKwh: f(20), EndKwh: f(60), PeakKw: 150, PowerKwh: f(90)}, nil, nil, Prices{})
	if !close(fast.WallKwh, 40) {
		t.Fatalf("%+v", fast.WallKwh)
	}
}

func TestANamedPlaceCanSetItsOwnPriceOrCountAsHome(t *testing.T) {
	cabin := PlaceRate{Name: "Cabin", Lat: 45.0, Lon: -90.0, RadiusM: 150, Price: f(0.20)}
	free := PlaceRate{Name: "Work", Lat: 46.0, Lon: -91.0, RadiusM: 150, Price: f(0)}
	house := PlaceRate{Name: "Parents", Lat: 47.0, Lon: -92.0, RadiusM: 150, Home: true}
	prices := Prices{Home: f(0.10), Public: f(0.50), Places: []PlaceRate{cabin, free, house}}
	at := func(lat, lon float64) Session {
		return Enrich(&store.Charge{Lat: f(lat), Lon: f(lon), StartKwh: f(10), EndKwh: f(19), PeakKw: 7}, nil, &Point{Lat: 40, Lon: -75}, prices) // 9 kWh, wall 10
	}
	if s := at(45.0001, -90.0001); !close(s.Cost, 2.0) || s.Where != "public" || s.Place != "Cabin" {
		t.Fatalf("the cabin has its own price: %+v %v", s.Where, s.Cost)
	}
	if s := at(46.0, -91.0); !close(s.Cost, 0) || s.Place != "Work" {
		t.Fatalf("free charging is a price of zero: %v", s.Cost)
	}
	if s := at(47.0, -92.0); !close(s.Cost, 1.0) || s.Where != "home" {
		t.Fatalf("a place that counts as home uses the home price: %+v %v", s.Where, s.Cost)
	}
	if s := at(10.0, 10.0); !close(s.Cost, 5.0) || s.Where != "public" || s.Place != "" {
		t.Fatalf("elsewhere is the usual public price: %+v %v", s.Where, s.Cost)
	}
	// a hand-set kind wins over the place
	hand := Enrich(&store.Charge{Kind: "home", Lat: f(45.0), Lon: f(-90.0), StartKwh: f(10), EndKwh: f(19), PeakKw: 7}, nil, nil, prices)
	if !close(hand.Cost, 1.0) {
		t.Fatalf("choosing home by hand uses the home price: %v", hand.Cost)
	}
}
