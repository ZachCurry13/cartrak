package server

import (
	"context"
	"time"

	"github.com/ZachCurry13/cartrak/internal/geocode"
)

// maxCitiesPerList is how many new places one view of the trip list asks the lookup service about. The service is slow on purpose, so a
// long history fills in a little each time the list is opened.
const maxCitiesPerList = 12

// nameCities fills in the city each trip started and ended in from what CarTrak has already looked up. Cities that are not known yet
// are looked up in the background (a few at a time, spaced out), so the list is never held up and shows them the next time.
func (s *Server) nameCities(ts []tripOut) {
	pl := s.places()
	var missing [][2]float64
	seen := map[string]bool{}
	name := func(lat, lon *float64) string {
		if lat == nil || lon == nil {
			return ""
		}
		if c, ok := pl.Cached(*lat, *lon); ok {
			return c
		}
		if k := geocode.Key(*lat, *lon); !seen[k] {
			seen[k] = true
			missing = append(missing, [2]float64{*lat, *lon})
		}
		return ""
	}
	for i := range ts {
		t := ts[i].Trip
		ts[i].StartCity = name(t.StartLat, t.StartLon)
		if !t.Open {
			ts[i].EndCity = name(t.EndLat, t.EndLon)
		}
	}
	if len(missing) == 0 || !s.CityLookups || (s.Cfg != nil && !s.Cfg.GeocodeEnabled()) || !s.cityBusy.CompareAndSwap(false, true) {
		return
	}
	if len(missing) > maxCitiesPerList {
		missing = missing[:maxCitiesPerList]
	}
	go func() {
		defer s.cityBusy.Store(false)
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		for _, p := range missing {
			if _, err := pl.Lookup(ctx, p[0], p[1]); err != nil {
				return // the service is refusing or unreachable: try again at the next view
			}
		}
	}()
}
