package server

import (
	"encoding/json"
	"sort"
	"sync"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/store"
	"github.com/ZachCurry13/cartrak/internal/trips"
)

// The car counts the energy braking gave back (tripXevBatteryChargeRegenerated) but does not say in what unit. One
// reading looked like hundredths of a kWh. Before showing it as the trip's regen energy, check on the owner's own recent
// trips that it agrees with the other figure the car gives: the range braking gave back, valued at the trip's use per
// kilometre. Only when most trips agree is the counter used.

const regenUnitKwh = 0.01

var (
	regenMu     sync.Mutex
	regenAt     time.Time
	regenAgrees bool
)

// regenRatiosAgree says whether the counter matches the estimate on most trips. ratios are counter / estimate.
func regenRatiosAgree(ratios []float64) bool {
	if len(ratios) < 3 {
		return false
	}
	good := 0
	for _, r := range ratios {
		if r >= 0.7 && r <= 1.4 {
			good++
		}
	}
	sorted := append([]float64(nil), ratios...)
	sort.Float64s(sorted)
	med := sorted[len(sorted)/2]
	return float64(good)/float64(len(ratios)) >= 0.8 && med >= 0.85 && med <= 1.18
}

// regenReadings is the two regen counters in the last reading of a trip.
func (s *Server) regenReadings(t *store.Trip) (rangeKm, counter *float64) {
	if t.EndTS == nil {
		return nil, nil
	}
	x, _ := s.S.SampleAt(*t.EndTS)
	if x == nil || x.TS < t.StartTS {
		return nil, nil
	}
	f := map[string]any{}
	if json.Unmarshal(x.Extra, &f) != nil {
		return nil, nil
	}
	return ford.Number(f, "tripXevBatteryRangeRegenerated"), ford.Number(f, "tripXevBatteryChargeRegenerated")
}

// regenCounterAgrees is the check above on the last dozen or so trips, kept for half an hour.
func (s *Server) regenCounterAgrees() bool {
	regenMu.Lock()
	defer regenMu.Unlock()
	if time.Since(regenAt) < 30*time.Minute {
		return regenAgrees
	}
	var ratios []float64
	list, _ := s.S.ListTrips(store.TripFilter{Limit: 20})
	for _, t := range list {
		if t.Open || t.EndTS == nil || t.DistanceKm < 3 {
			continue
		}
		e, has := trips.EnergyUsed(t, s.capacityKwh())
		if !has || e.Kwh < 0.5 {
			continue
		}
		rk, c := s.regenReadings(t)
		if rk == nil || c == nil || *rk < 0.5 || *c <= 0 {
			continue
		}
		est := *rk * e.Kwh / t.DistanceKm
		ratios = append(ratios, *c*regenUnitKwh/est)
	}
	regenAgrees, regenAt = regenRatiosAgree(ratios), time.Now()
	return regenAgrees
}
