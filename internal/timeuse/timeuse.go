// Package timeuse works out how a stretch of time was spent: driving, charging, or parked, day by day and by
// where the car was parked. It uses the recorded trips and charging sessions, so it needs no extra data.
package timeuse

import (
	"sort"
	"time"

	"github.com/ZachCurry13/cartrak/internal/places"
	"github.com/ZachCurry13/cartrak/internal/store"
)

// Span is a start and an end in unix seconds.
type Span struct{ From, To int64 }

// Day is one local day.
type Day struct {
	Date     string `json:"date"` // 2026-10-13
	Driving  int64  `json:"driving"`
	Charging int64  `json:"charging"`
	Parked   int64  `json:"parked"`
}

// PlaceTime is how long the car was parked at a named place.
type PlaceTime struct {
	Name string `json:"name"`
	Sec  int64  `json:"sec"`
}

type Report struct {
	From     int64       `json:"from"`
	To       int64       `json:"to"`
	Driving  int64       `json:"driving"`
	Charging int64       `json:"charging"`
	Parked   int64       `json:"parked"`
	Days     []Day       `json:"days"`
	Places   []PlaceTime `json:"places"` // parked time by named place, biggest first, with "Elsewhere" last
}

func clip(s Span, from, to int64) (Span, bool) {
	if s.From < from {
		s.From = from
	}
	if s.To > to {
		s.To = to
	}
	return s, s.To > s.From
}

func sum(spans []Span, from, to int64) int64 {
	var n int64
	for _, s := range spans {
		if c, ok := clip(s, from, to); ok {
			n += c.To - c.From
		}
	}
	return n
}

// merge joins overlapping spans so time is never counted twice.
func merge(in []Span) []Span {
	s := append([]Span(nil), in...)
	sort.Slice(s, func(i, j int) bool { return s[i].From < s[j].From })
	var out []Span
	for _, x := range s {
		if n := len(out); n > 0 && x.From <= out[n-1].To {
			if x.To > out[n-1].To {
				out[n-1].To = x.To
			}
		} else {
			out = append(out, x)
		}
	}
	return out
}

func tripSpans(ts []*store.Trip, now int64) []Span {
	var out []Span
	for _, t := range ts {
		end := now
		if t.EndTS != nil {
			end = *t.EndTS
		}
		if end > t.StartTS {
			out = append(out, Span{t.StartTS, end})
		}
	}
	return merge(out)
}

func chargeSpans(cs []*store.Charge, now int64) []Span {
	var out []Span
	for _, c := range cs {
		end := now
		if c.EndTS != nil {
			end = *c.EndTS
		}
		if end > c.StartTS {
			out = append(out, Span{c.StartTS, end})
		}
	}
	return merge(out)
}

// subtract removes the spans in cut from a (all merged and sorted), for example charging time that overlaps driving.
func subtract(a, cut []Span) []Span {
	var out []Span
	for _, s := range a {
		cur := s
		for _, c := range cut {
			if c.To <= cur.From || c.From >= cur.To {
				continue
			}
			if c.From > cur.From {
				out = append(out, Span{cur.From, c.From})
			}
			cur.From = c.To
			if cur.From >= cur.To {
				break
			}
		}
		if cur.To > cur.From {
			out = append(out, cur)
		}
	}
	return out
}

// Analyze covers from..to (unix seconds). `first` is when CarTrak's first reading was, so time before CarTrak
// existed is not counted as parked. ts must be in start order.
func Analyze(ts []*store.Trip, cs []*store.Charge, named []*store.NamedPlace, from, to, first int64, loc *time.Location) Report {
	if first > from {
		from = first
	}
	rep := Report{From: from, To: to, Days: []Day{}, Places: []PlaceTime{}}
	if to <= from {
		return rep
	}
	drive := tripSpans(ts, to)
	charge := subtract(chargeSpans(cs, to), drive)
	rep.Driving, rep.Charging = sum(drive, from, to), sum(charge, from, to)
	rep.Parked = to - from - rep.Driving - rep.Charging
	if rep.Parked < 0 {
		rep.Parked = 0
	}

	// By day.
	day := time.Unix(from, 0).In(loc)
	day = time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, loc)
	for d := day; d.Unix() < to; d = d.AddDate(0, 0, 1) {
		a, b := d.Unix(), d.AddDate(0, 0, 1).Unix()
		if a < from {
			a = from
		}
		if b > to {
			b = to
		}
		dr, ch := sum(drive, a, b), sum(charge, a, b)
		rep.Days = append(rep.Days, Day{Date: d.Format("2006-01-02"), Driving: dr, Charging: ch, Parked: max(0, b-a-dr-ch)})
	}

	// Parked time by place: each stay is where the previous trip ended, from its end to the next trip's start
	// (or now), less any charging within it.
	byPlace := map[string]int64{}
	for i, t := range ts {
		if t.EndTS == nil {
			continue
		}
		stayEnd := to
		if i+1 < len(ts) {
			stayEnd = ts[i+1].StartTS
		}
		stay := Span{*t.EndTS, stayEnd}
		if c, ok := clip(stay, from, to); ok {
			secs := c.To - c.From - sum(charge, c.From, c.To)
			if secs <= 0 {
				continue
			}
			name := "Elsewhere"
			if n := places.Name(t.EndLat, t.EndLon, named); n != "" {
				name = n
			}
			byPlace[name] += secs
		}
	}
	for name, sec := range byPlace {
		if name != "Elsewhere" {
			rep.Places = append(rep.Places, PlaceTime{name, sec})
		}
	}
	sort.Slice(rep.Places, func(i, j int) bool { return rep.Places[i].Sec > rep.Places[j].Sec })
	if sec := byPlace["Elsewhere"]; sec > 0 {
		rep.Places = append(rep.Places, PlaceTime{"Elsewhere", sec})
	}
	return rep
}
