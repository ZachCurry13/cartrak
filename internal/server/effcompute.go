package server

import (
	"github.com/ZachCurry13/cartrak/internal/savings"
	"github.com/ZachCurry13/cartrak/internal/store"
)

// efficiencyFor builds the car's distance-per-kWh history from its finished trips.
func (s *Server) efficiencyFor() *savings.Efficiency {
	if e := s.effKept.get(); e != nil {
		return e
	}
	e := s.computeEfficiency()
	s.effKept.put(e)
	return e
}

func (s *Server) computeEfficiency() *savings.Efficiency {
	ts, err := s.S.ListTrips(store.TripFilter{All: true})
	if err != nil {
		return savings.NewEfficiency(nil)
	}
	capacity := s.capacityKwh()
	var ds []savings.Drive
	for _, t := range ts {
		if t.Open || t.EndTS == nil {
			continue
		}
		if o := withEnergy(t, capacity); o.EnergyKwh != nil {
			ds = append(ds, savings.Drive{TS: t.StartTS, Km: t.DistanceKm, Kwh: *o.EnergyKwh})
		}
	}
	return savings.NewEfficiency(ds)
}
