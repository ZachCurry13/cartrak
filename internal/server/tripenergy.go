package server

import (
	"github.com/ZachCurry13/cartrak/internal/store"
	"github.com/ZachCurry13/cartrak/internal/trips"
)

// tripOut is a trip with the energy it used. The page turns energy and distance into miles per
// kWh (or kWh per 100 km), so it follows each device's unit choice.
type tripOut struct {
	*store.Trip
	EnergyKwh       *float64 `json:"energyKwh"`
	EnergyEstimated bool     `json:"energyEstimated"` // from battery percent, not the car's own energy reading
}

// capacityKwh is the battery's latest estimated capacity, used to estimate trips that were recorded
// without the car's energy reading.
func (s *Server) capacityKwh() *float64 {
	if x, _ := s.S.LatestSample(); x != nil {
		return x.CapacityKwh
	}
	return nil
}

func withEnergy(t *store.Trip, capacity *float64) tripOut {
	out := tripOut{Trip: t}
	if t.Open {
		return out
	}
	if e, ok := trips.EnergyUsed(t, capacity); ok {
		kwh := e.Kwh
		out.EnergyKwh, out.EnergyEstimated = &kwh, e.Estimated
	}
	return out
}

func (s *Server) tripsWithEnergy(ts []*store.Trip) []tripOut {
	capacity := s.capacityKwh()
	out := make([]tripOut, 0, len(ts))
	for _, t := range ts {
		out = append(out, withEnergy(t, capacity))
	}
	return out
}
