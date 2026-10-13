package mileage

import (
	"math"
	"sort"

	"github.com/ZachCurry13/cartrak/internal/store"
)

// The car's odometer only moves when the car does, so the difference between two readings is exact even
// if CarTrak missed every trip in between. CheckTrips lays the recorded trips over that distance and
// reports the stretches no trip covers.
const (
	minPairKm = 0.3  // smaller steps are rounding in the odometer
	minGapKm  = 1.0  // a gap must add up to at least this to be reported
	okKm      = 1.6  // about a mile: close enough to call it a match
	edgeMaxKm = 8.0  // a small stretch right next to a recorded trip is its start or end, not a missed trip
	touchKm   = 0.05 // odometer values this close are the same point
)

// Gap is a stretch of driving that no trip covers.
type Gap struct {
	FromTS   int64   `json:"fromTs"` // the reading before the stretch
	ToTS     int64   `json:"toTs"`   // the reading after it
	StartOdo float64 `json:"startOdo"`
	EndOdo   float64 `json:"endOdo"`
	Km       float64 `json:"km"`
	// Edge is "start" or "end" when the gap sits right before or after one recorded trip (TripID),
	// which means that trip was picked up late or closed late. It is empty for a missed trip.
	Edge   string `json:"edge,omitempty"`
	TripID int64  `json:"tripId,omitempty"`
}

// Check compares the car's odometer with the recorded trips.
type Check struct {
	Readings  int     `json:"readings"`
	DrivenKm  float64 `json:"drivenKm"`
	CoveredKm float64 `json:"coveredKm"`
	MissingKm float64 `json:"missingKm"`
	Ok        bool    `json:"ok"`
	Gaps      []Gap   `json:"gaps"`
}

type span struct{ lo, hi float64 }

// covered merges the trips' odometer ranges.
func covered(trips []*store.Trip) []span {
	var s []span
	for _, t := range trips {
		if t.StartOdo != nil && t.EndOdo != nil && *t.EndOdo > *t.StartOdo {
			s = append(s, span{*t.StartOdo, *t.EndOdo})
		}
	}
	sort.Slice(s, func(i, j int) bool { return s[i].lo < s[j].lo })
	var out []span
	for _, x := range s {
		if n := len(out); n > 0 && x.lo <= out[n-1].hi {
			out[n-1].hi = math.Max(out[n-1].hi, x.hi)
		} else {
			out = append(out, x)
		}
	}
	return out
}

func overlap(lo, hi float64, spans []span) float64 {
	var sum float64
	for _, s := range spans {
		sum += math.Max(0, math.Min(hi, s.hi)-math.Max(lo, s.lo))
	}
	return sum
}

// CheckTrips takes the odometer readings of the period in time order (the first is where the car stood at
// the start) and the trips that overlap them.
func CheckTrips(pts []store.OdoPoint, trips []*store.Trip) Check {
	c := Check{Readings: len(pts), Gaps: []Gap{}}
	if len(pts) < 2 {
		c.Ok = true
		return c
	}
	spans := covered(trips)
	var open *Gap
	flush := func() {
		if open != nil && open.Km >= minGapKm {
			c.Gaps = append(c.Gaps, *open)
		}
		open = nil
	}
	for i := 1; i < len(pts); i++ {
		a, b := pts[i-1], pts[i]
		delta := b.Km - a.Km
		if delta < minPairKm {
			flush() // parked, or noise: a gap cannot continue across it
			continue
		}
		c.DrivenKm += delta
		missing := delta - overlap(a.Km, b.Km, spans)
		if missing < minPairKm {
			flush()
			continue
		}
		if open == nil {
			open = &Gap{FromTS: a.TS, StartOdo: a.Km}
		}
		open.ToTS, open.EndOdo = b.TS, b.Km
		open.Km += missing
	}
	flush()
	for i := range c.Gaps {
		c.MissingKm += c.Gaps[i].Km
		markEdge(&c.Gaps[i], trips)
	}
	c.CoveredKm = c.DrivenKm - c.MissingKm
	c.Ok = c.MissingKm < okKm
	return c
}

// markEdge notes a small gap that touches exactly one recorded trip.
func markEdge(g *Gap, trips []*store.Trip) {
	var before, after *store.Trip // the trip that ends where the gap starts, and the one that starts where it ends
	for _, t := range trips {
		if t.StartOdo == nil || t.EndOdo == nil {
			continue
		}
		if math.Abs(*t.EndOdo-g.StartOdo) < touchKm {
			before = t
		}
		if math.Abs(*t.StartOdo-g.EndOdo) < touchKm {
			after = t
		}
	}
	if g.Km > edgeMaxKm || (before == nil) == (after == nil) {
		return
	}
	if before != nil {
		g.Edge, g.TripID = "end", before.ID
	} else {
		g.Edge, g.TripID = "start", after.ID
	}
}
