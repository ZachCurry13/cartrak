// Package stops finds the stretches a car sat parked between two trips, such as time spent in a customer's parking lot. They
// are worked out from the trips already recorded, so earlier days have them too; only a person's label and note are stored.
package stops

import (
	"sort"
)

const (
	// MinSec is the shortest gap that counts as a stop; shorter ones are red lights and quick errands.
	MinSec = 10 * 60
	// MaxSec is the longest: a car that sat for more than 8 hours was parked, not worked in.
	MaxSec = 8 * 60 * 60
	// MinIdleSec is how long the car must have sat still, engine on, before a trip ended for that wait to count as part of the stop.
	MinIdleSec = 2 * 60
)

// Trip is what a stop needs to know about a trip.
type Trip struct {
	ID       int64
	StartTS  int64
	EndTS    int64 // 0 while the trip is still going
	EndLat   *float64
	EndLon   *float64
	Category string
	// LastMoving is when the car was last seen moving before the trip ended, 0 when unknown. A driver who waits with the engine on
	// ends the trip only when it is switched off; the wait belongs to the stop, so the stop begins here.
	LastMoving int64
}

// Label is what a person said about a stop, kept against the trip that arrived there.
type Label struct {
	Category string // "" when only a note was written
	Note     string
}

// Stop is one stretch of parked time.
type Stop struct {
	TripID   int64    `json:"tripId"` // the trip that arrived here; it identifies the stop
	Start    int64    `json:"start"`
	End      int64    `json:"end"`
	Minutes  int      `json:"minutes"`
	Lat      *float64 `json:"lat"`
	Lon      *float64 `json:"lon"`
	Place    string   `json:"place"` // filled in by the caller: a named place or a town
	Category string   `json:"category"`
	Manual   bool     `json:"manual"` // the category was chosen by a person, not taken from the trip that arrived
	Note     string   `json:"note"`
	Open     bool     `json:"open"` // the car is still parked here
}

// Skip says whether a place or a stretch of time is not a stop: at home, or while the car was charging.
type Skip func(lat, lon *float64, start, end int64) bool

// Build lists the stops after each finished trip, oldest first. now closes the last one while the car is still parked.
func Build(trips []Trip, labels map[int64]Label, skip Skip, now int64) []Stop {
	ts := append([]Trip(nil), trips...)
	sort.Slice(ts, func(i, j int) bool { return ts[i].StartTS < ts[j].StartTS })
	var out []Stop
	for i, t := range ts {
		if t.EndTS == 0 {
			continue
		}
		end, open := now, true
		if i+1 < len(ts) {
			end, open = ts[i+1].StartTS, false
		}
		from := t.EndTS
		if t.LastMoving > t.StartTS && t.LastMoving <= t.EndTS-MinIdleSec {
			from = t.LastMoving
		}
		gap := end - from
		if gap < MinSec || gap > MaxSec || (skip != nil && skip(t.EndLat, t.EndLon, from, end)) {
			continue
		}
		s := Stop{TripID: t.ID, Start: from, End: end, Minutes: int(gap / 60), Lat: t.EndLat, Lon: t.EndLon, Category: t.Category, Open: open}
		if l, ok := labels[t.ID]; ok {
			s.Note = l.Note
			if l.Category != "" {
				s.Category, s.Manual = l.Category, true
			}
		}
		out = append(out, s)
	}
	return out
}

// Day is one calendar day's worth of time in the work category.
type Day struct {
	Date    string `json:"date"`
	Minutes int    `json:"minutes"` // all stops that day
	Work    int    `json:"work"`    // of them, work stops
}

// Totals adds up stop minutes by the local day each stop began on.
func Totals(ss []Stop, dayOf func(ts int64) string) []Day {
	by := map[string]*Day{}
	for _, s := range ss {
		k := dayOf(s.Start)
		d := by[k]
		if d == nil {
			d = &Day{Date: k}
			by[k] = d
		}
		d.Minutes += s.Minutes
		if s.Category == "work" {
			d.Work += s.Minutes
		}
	}
	out := make([]Day, 0, len(by))
	for _, d := range by {
		out = append(out, *d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Date > out[j].Date })
	return out
}
