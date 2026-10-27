package trips

import (
	"strconv"
	"strings"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

// WorkDay is the rule "the first outing of a work day is work". An outing is everything from leaving Home until the car is back
// there, whatever the clock says; after it, the day's later outings are personal unless the owner says otherwise. A trip that
// continues an outing (the car is not at Home between the two) takes the category of the trip before it.
type WorkDay struct {
	On      bool
	Days    int   // the weekdays that are work days, Sunday = bit 1, as in a rule
	StaySec int64 // a stay at Home shorter than this (lunch) does not end the work day
}

const (
	outingGapSec   = 12 * 3600 // a stop this long away from Home starts a new outing
	priorWindowSec = 36 * 3600 // how far back the earlier trips are read
)

type spot struct{ lat, lon, r float64 }

// isHomeName is a place named Home, or Home and a number for a second one ("Home 2").
func isHomeName(n string) bool {
	f := strings.Fields(strings.ToLower(n))
	switch len(f) {
	case 1:
		return f[0] == "home"
	case 2:
		_, err := strconv.Atoi(f[1])
		return f[0] == "home" && err == nil
	}
	return false
}

func homeSpots(ps []*store.NamedPlace) []spot {
	var out []spot
	for _, p := range ps {
		if isHomeName(p.Name) {
			r := p.RadiusM
			if r <= 0 {
				r = 150
			}
			out = append(out, spot{p.Lat, p.Lon, r})
		}
	}
	return out
}

// atHome says whether a position is within a Home place; known is false when the car gave no position.
func atHome(spots []spot, lat, lon *float64) (known, home bool) {
	if lat == nil || lon == nil {
		return false, false
	}
	for _, s := range spots {
		if Haversine(*lat, *lon, s.lat, s.lon) <= s.r {
			return true, true
		}
	}
	return true, false
}

// workDayCategory decides a trip's category from the trips before it (oldest first). ok is false when the rule has nothing to say.
func workDayCategory(w WorkDay, spots []spot, loc *time.Location, prior []*store.Trip, startTS int64) (cat string, ok bool) {
	if !w.On || len(spots) == 0 {
		return "", false
	}
	var last *store.Trip
	for i := len(prior) - 1; i >= 0; i-- {
		if prior[i].StartTS < startTS {
			last = prior[i]
			break
		}
	}
	if last != nil && last.EndTS != nil && startTS-*last.EndTS < outingGapSec {
		known, home := atHome(spots, last.EndLat, last.EndLon)
		if !known {
			return "", false
		}
		if !home {
			return last.Category, true // still out: the same outing
		}
	}
	at := time.Unix(startTS, 0).In(loc)
	if w.Days&(1<<uint(at.Weekday())) == 0 {
		return "", false
	}
	midnight := time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, loc).Unix()
	earlier := false
	for _, p := range prior {
		if p.StartTS >= midnight && p.StartTS < startTS {
			earlier = true
			break
		}
	}
	switch {
	case !earlier:
		return Work, true
	case last != nil && last.EndTS != nil && *last.EndTS >= midnight && last.Category == Work && startTS-*last.EndTS < w.StaySec:
		return Work, true // home only briefly: the work day goes on
	}
	return Personal, true
}

// priorTrips are the trips in the day and a half before ts, oldest first, read from the store.
func (d *Detector) priorTrips(ts int64) []*store.Trip {
	ps, err := d.S.ListTrips(store.TripFilter{From: ts - priorWindowSec, To: ts, All: true})
	if err != nil {
		return nil
	}
	for i, j := 0, len(ps)-1; i < j; i, j = i+1, j-1 {
		ps[i], ps[j] = ps[j], ps[i]
	}
	return ps
}
