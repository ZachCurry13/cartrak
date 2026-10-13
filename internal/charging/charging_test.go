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
