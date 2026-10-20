package server

import (
	"context"
	"time"

	"github.com/ZachCurry13/cartrak/internal/geocode"
	"github.com/ZachCurry13/cartrak/internal/mileage"
)

// maxCitiesPerList is how many new places one view of a list asks the lookup service about. The service is slow on purpose, so a
// long history fills in a little each time the list is opened.
const maxCitiesPerList = 12

// cityNamer names places from what CarTrak has already looked up, and remembers the ones it does not know yet so they can be
// looked up in the background (a few at a time, spaced out): a page is never held up and shows them the next time.
type cityNamer struct {
	s       *Server
	pl      mileage.Places
	missing [][2]float64
	seen    map[string]bool
}

func (s *Server) cityNamer() *cityNamer {
	return &cityNamer{s: s, pl: s.places(), seen: map[string]bool{}}
}

// name is the city at a point, or "" when it is not known yet.
func (c *cityNamer) name(lat, lon *float64) string {
	if lat == nil || lon == nil {
		return ""
	}
	if city, ok := c.pl.Cached(*lat, *lon); ok {
		return city
	}
	if k := geocode.Key(*lat, *lon); !c.seen[k] {
		c.seen[k] = true
		c.missing = append(c.missing, [2]float64{*lat, *lon})
	}
	return ""
}

// lookupLater asks about the unknown places in the background, unless lookups are off or one is already running.
func (c *cityNamer) lookupLater() {
	s, missing := c.s, c.missing
	if len(missing) == 0 || !s.CityLookups || (s.Cfg != nil && !s.Cfg.GeocodeEnabled()) || !s.cityBusy.CompareAndSwap(false, true) {
		return
	}
	if len(missing) > maxCitiesPerList {
		missing = missing[:maxCitiesPerList]
	}
	pl := c.pl
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

// nameCities fills in the city each trip started and ended in.
func (s *Server) nameCities(ts []tripOut) {
	c := s.cityNamer()
	for i := range ts {
		t := ts[i].Trip
		ts[i].StartCity = c.name(t.StartLat, t.StartLon)
		if !t.Open {
			ts[i].EndCity = c.name(t.EndLat, t.EndLon)
		}
	}
	c.lookupLater()
}
