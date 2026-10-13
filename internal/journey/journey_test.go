package journey

import (
	"math"
	"testing"

	"github.com/ZachCurry13/cartrak/internal/charging"
	"github.com/ZachCurry13/cartrak/internal/store"
)

func f(v float64) *float64 { return &v }

func leg(start, end int64, km, kwhFrom, kwhTo float64) *store.Trip {
	return &store.Trip{StartTS: start, EndTS: &end, DistanceKm: km, StartKwh: f(kwhFrom), EndKwh: f(kwhTo), EndLat: f(40), EndLon: f(-75)}
}

func TestAJourneyAddsUpLegsStopsEnergyAndCost(t *testing.T) {
	ts := []*store.Trip{leg(1000, 4600, 200, 80, 50), leg(6400, 10000, 150, 78, 56)}
	// A charge during the stop (parked from 4600 to 6400) added 28 kWh at 0.50 a kWh, wall energy 31.1.
	cost := 15.0
	sessions := []charging.Session{{Charge: &store.Charge{StartTS: 4700}, AddedKwh: f(28), Cost: &cost}}
	s := Summarize(ts, nil, sessions, f(0.1))
	if s.Trips != 2 || s.DistanceKm != 350 || s.MovingSec != 7200 || s.ElapsedSec != 9000 {
		t.Fatalf("%+v", s)
	}
	if s.KwhUsed == nil || math.Abs(*s.KwhUsed-52) > 1e-9 || s.KwhEstimated || math.Abs(*s.KmPerKwh-350.0/52) > 1e-9 {
		t.Fatalf("energy: %+v", s)
	}
	if len(s.Stops) != 1 || !s.Stops[0].Charging || s.Stops[0].Seconds != 1800 || *s.Stops[0].AddedKwh != 28 || s.ChargedKwh != 28 {
		t.Fatalf("stops: %+v", s.Stops)
	}
	if s.Cost == nil || *s.Cost != 15 || s.CostEstimate {
		t.Fatalf("a priced charging session is the cost: %+v", s.Cost)
	}
}

func TestWithoutPricedChargesTheCostIsAnEstimateFromEnergyUsed(t *testing.T) {
	ts := []*store.Trip{leg(0, 3600, 100, 80, 70)}
	s := Summarize(ts, nil, nil, f(0.20))
	if s.Cost == nil || !s.CostEstimate || math.Abs(*s.Cost-10/0.9*0.2) > 1e-9 {
		t.Fatalf("%+v", s.Cost)
	}
	if none := Summarize(ts, nil, nil, nil); none.Cost != nil {
		t.Fatal("no price, no cost")
	}
	open := &store.Trip{StartTS: 5, Open: true}
	if s := Summarize([]*store.Trip{open}, nil, nil, nil); s.Trips != 0 {
		t.Fatal("a trip still in progress is not counted")
	}
}
