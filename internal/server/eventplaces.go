package server

import (
	"github.com/ZachCurry13/cartrak/internal/history"
	"github.com/ZachCurry13/cartrak/internal/places"
	"github.com/ZachCurry13/cartrak/internal/store"
	"github.com/ZachCurry13/cartrak/internal/trips"
)

// placeEvents fills in where each event of the Day view and History happened when no named place covers it: the city from
// CarTrak's lookups (new places are looked up in the background, like the trip list), a drive's start and end, and "around" a
// city for a drive that began and ended in it. Drives also get the energy they used, for their efficiency.
func (s *Server) placeEvents(evs []history.Event, ts []*store.Trip, named []*store.NamedPlace) {
	byID := make(map[int64]*store.Trip, len(ts))
	for _, t := range ts {
		byID[t.ID] = t
	}
	capacity := s.capacityKwh()
	cities := s.cityNamer()
	for i := range evs {
		e := &evs[i]
		if e.Kind != "driving" {
			if e.Place == "" {
				e.Place = cities.name(e.Lat, e.Lon)
			}
			continue
		}
		t := byID[e.ID]
		if t == nil {
			continue
		}
		if en, ok := trips.EnergyUsed(t, capacity); ok && !t.Open {
			kwh := en.Kwh
			e.EnergyKwh = &kwh
		}
		// a place you named wins, otherwise the city; Build leaves the start empty when it matches the end, so it is worked out again
		from, to := places.Name(t.StartLat, t.StartLon, named), e.Place
		if from == "" {
			from = cities.name(t.StartLat, t.StartLon)
		}
		if to == "" && !t.Open {
			to = cities.name(t.EndLat, t.EndLon)
		}
		if from != "" && from == to { // a drive that began and ended in the same place is shown once, as "around" it
			from, to = "", "Around "+to
		}
		e.FromPlace, e.Place = from, to
	}
	cities.lookupLater()
}
