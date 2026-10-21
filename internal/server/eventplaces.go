package server

import (
	"time"

	"github.com/ZachCurry13/cartrak/internal/history"
	"github.com/ZachCurry13/cartrak/internal/places"
	"github.com/ZachCurry13/cartrak/internal/store"
	"github.com/ZachCurry13/cartrak/internal/trips"
)

// placeEvents fills in where each event of the Day view and History happened when no named place covers it: the city from
// CarTrak's lookups (new places are looked up in the background, like the trip list), a drive's start and end, and "around" a
// city for a drive that began and ended in it. Drives also get the energy they used, for their efficiency.
func (s *Server) placeEvents(evs []history.Event, ts []*store.Trip, named []*store.NamedPlace, bases *baseFinder) {
	byID := make(map[int64]*store.Trip, len(ts))
	for _, t := range ts {
		byID[t.ID] = t
	}
	capacity := s.capacityKwh()
	cities := bases.cities
	baseAt := func(ts int64) *tripBase { return bases.forDay(time.Unix(ts, 0).In(bases.loc).Format("2006-01-02")) }
	for i := range evs {
		e := &evs[i]
		if e.Kind != "driving" {
			if b := baseAt(e.Start); b.near(e.Lat, e.Lon) {
				e.Place, e.Base = b.label(), e.Kind == "parked"
			} else if e.Place == "" {
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
		b := baseAt(t.StartTS)
		switch atStart, atEnd := b.near(t.StartLat, t.StartLon), b.near(t.EndLat, t.EndLon); {
		case atStart && !atEnd:
			from = "Base"
		case atEnd && !atStart && !t.Open:
			to = "Base"
		}
		if from != "" && from == to { // a drive that began and ended in the same place is shown once, as "around" it
			from, to = "", "Around "+to
		}
		e.FromPlace, e.Place = from, to
	}
}
