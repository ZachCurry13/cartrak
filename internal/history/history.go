// Package history merges trips and charging sessions into one feed, newest first, and fills the gaps between them with
// the time the car was parked.
package history

import (
	"sort"

	"github.com/ZachCurry13/cartrak/internal/places"
	"github.com/ZachCurry13/cartrak/internal/store"
)

// Charge is a charging session with what the server worked out about it.
type Charge struct {
	ID       int64
	Start    int64
	End      *int64
	StartSoC *float64
	EndSoC   *float64
	Lat, Lon *float64
	AddedKwh *float64
	Cost     *float64
	Where    string // "home", "public" or ""
	// CostActual says the cost is what was paid (a receipt), not worked out from a price; WallKwh is what the charger delivered.
	CostActual bool
	WallKwh    *float64
}

// Event is one row of the feed.
type Event struct {
	Kind       string   `json:"kind"` // driving, charging, parked or remote (a remote start)
	ID         int64    `json:"id"`   // trip or session id; 0 for parked
	Start      int64    `json:"start"`
	End        *int64   `json:"end"` // nil while it is still going
	StartSoC   *float64 `json:"startSoc"`
	EndSoC     *float64 `json:"endSoc"`
	DistanceKm float64  `json:"distanceKm"`
	AddedKwh   *float64 `json:"addedKwh"`
	Cost       *float64 `json:"cost"`
	Place      string   `json:"place"`     // where it happened (the end, for a drive)
	FromPlace  string   `json:"fromPlace"` // where a drive began, when it differs
	Where      string   `json:"where"`     // charging: home, public or ""
	Lat        *float64 `json:"lat"`       // where it happened (the end of a drive), for a map
	Lon        *float64 `json:"lon"`
	EnergyKwh  *float64 `json:"energyKwh,omitempty"`  // a drive's energy use, for its efficiency
	Base       bool     `json:"base,omitempty"`       // at the base of a trip away: not counted as a stop
	CostActual bool     `json:"costActual,omitempty"` // a charge's cost is from a receipt
	WallKwh    *float64 `json:"wallKwh,omitempty"`    // what the charger delivered, for the price per kWh
	TempC      *float64 `json:"tempC,omitempty"`      // a remote start: the temperature outside
	Restarts   int      `json:"restarts,omitempty"`   // a remote start: times it was started again
	Cont       bool     `json:"cont,omitempty"`       // parked: the rest of a stop that a remote start split, not a stop of its own
}

// minParked is the shortest gap worth showing as its own row.
const minParked = 5 * 60

type span struct {
	e   Event
	lat *float64
	lon *float64
}

// Build is every trip and charging session that overlaps from..to, plus the parked time between them, newest first.
// now ends a parked stretch that is still going.
func Build(trips []*store.Trip, charges []Charge, named []*store.NamedPlace, from, to, now int64) []Event {
	var all []span
	for _, t := range trips {
		e := Event{Kind: "driving", ID: t.ID, Start: t.StartTS, End: t.EndTS, StartSoC: t.StartSoC, EndSoC: t.EndSoC, DistanceKm: t.DistanceKm,
			Place: places.Name(t.EndLat, t.EndLon, named), FromPlace: places.Name(t.StartLat, t.StartLon, named)}
		if e.FromPlace == e.Place {
			e.FromPlace = ""
		}
		e.Lat, e.Lon = t.EndLat, t.EndLon
		all = append(all, span{e, t.EndLat, t.EndLon})
	}
	for _, c := range charges {
		e := Event{Kind: "charging", ID: c.ID, Start: c.Start, End: c.End, StartSoC: c.StartSoC, EndSoC: c.EndSoC, AddedKwh: c.AddedKwh, Cost: c.Cost,
			Place: places.Name(c.Lat, c.Lon, named), Where: c.Where, CostActual: c.CostActual, WallKwh: c.WallKwh}
		e.Lat, e.Lon = c.Lat, c.Lon
		all = append(all, span{e, c.Lat, c.Lon})
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].e.Start < all[j].e.Start })

	var out []Event
	var prev *span
	var cursor int64
	for i := range all {
		cur := &all[i]
		if prev != nil && cur.e.Start-cursor >= minParked {
			out = append(out, parked(prev, cursor, cur.e.Start, cur.e.StartSoC, named))
		}
		out = append(out, cur.e)
		end := now // still going: nothing can be parked after it
		if cur.e.End != nil {
			end = *cur.e.End
		}
		if end > cursor {
			cursor = end
		}
		prev = cur
	}
	if prev != nil && prev.e.End != nil && now-cursor >= minParked {
		out = append(out, parked(prev, cursor, 0, nil, named))
	}
	var keep []Event
	for _, e := range out {
		end := now
		if e.End != nil {
			end = *e.End
		}
		if end >= from && e.Start <= to {
			keep = append(keep, e)
		}
	}
	sort.SliceStable(keep, func(i, j int) bool { return keep[i].Start > keep[j].Start })
	if keep == nil {
		keep = []Event{}
	}
	return keep
}

// parked is the gap after prev. until is 0 for a stretch that is still going.
func parked(prev *span, from, until int64, nextSoC *float64, named []*store.NamedPlace) Event {
	e := Event{Kind: "parked", Start: from, StartSoC: prev.e.EndSoC, EndSoC: nextSoC, Place: prev.e.Place, Lat: prev.lat, Lon: prev.lon}
	if e.Place == "" && prev.lat != nil {
		e.Place = places.Name(prev.lat, prev.lon, named)
	}
	if until > 0 {
		e.End = &until
	}
	return e
}

// Totals is what the events of a period add up to. Seconds are clipped to the period.
type Totals struct {
	Kwh         float64 `json:"kwh"`
	Km          float64 `json:"km"`
	Cost        float64 `json:"cost"`
	ParkedSec   int64   `json:"parkedSec"`
	DrivingSec  int64   `json:"drivingSec"`
	ChargingSec int64   `json:"chargingSec"`
}

// Sum adds up events that fall in from..to.
func Sum(events []Event, from, to, now int64) Totals {
	var t Totals
	for _, e := range events {
		s, end := e.Start, now
		if e.End != nil {
			end = *e.End
		}
		if s < from {
			s = from
		}
		if end > to {
			end = to
		}
		sec := end - s
		if sec < 0 {
			sec = 0
		}
		switch e.Kind {
		case "driving":
			t.Km += e.DistanceKm
			t.DrivingSec += sec
		case "charging":
			if e.AddedKwh != nil {
				t.Kwh += *e.AddedKwh
			}
			if e.Cost != nil {
				t.Cost += *e.Cost
			}
			t.ChargingSec += sec
		case "parked":
			t.ParkedSec += sec
		}
	}
	return t
}
