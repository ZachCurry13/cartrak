// Package periodstats adds driving and charging up by day, month or year for a bar chart, and splits charging into
// home, work and public.
package periodstats

import (
	"strings"
	"time"
)

// Trip and Charge are the little each calculation needs.
type Trip struct {
	Start      int64
	DistanceKm float64
	Seconds    int64
}

type Charge struct {
	Start    int64
	AddedKwh float64
	Cost     float64
	Seconds  int64
	Kind     string // home, work or public
}

// Bucket is one bar: a day, a month or a year.
type Bucket struct {
	Label       string  `json:"label"`
	Start       int64   `json:"start"`
	Km          float64 `json:"km"`
	Kwh         float64 `json:"kwh"`
	Cost        float64 `json:"cost"`
	DrivingSec  int64   `json:"drivingSec"`
	ChargingSec int64   `json:"chargingSec"`
}

// Kind is the share of charging at one kind of place.
type Kind struct {
	Kind     string  `json:"kind"`
	Kwh      float64 `json:"kwh"`
	Cost     float64 `json:"cost"`
	Sessions int     `json:"sessions"`
}

// Report is the bars and the split for the same stretch.
type Report struct {
	Period  string   `json:"period"`
	From    int64    `json:"from"`
	To      int64    `json:"to"`
	Buckets []Bucket `json:"buckets"`
	Kinds   []Kind   `json:"kinds"`
}

// Range is the stretch a period covers around at: a day bar chart covers the month, a month bar chart the year, and
// a year bar chart every year from first on (first is the earliest time anything was recorded, or 0).
func Range(period string, at time.Time, first int64, now time.Time) (from, to time.Time, ok bool) {
	loc := at.Location()
	switch period {
	case "day":
		from = time.Date(at.Year(), at.Month(), 1, 0, 0, 0, 0, loc)
		return from, from.AddDate(0, 1, 0), true
	case "month":
		from = time.Date(at.Year(), 1, 1, 0, 0, 0, 0, loc)
		return from, from.AddDate(1, 0, 0), true
	case "year":
		start := now.In(loc).Year() - 4
		if first > 0 {
			if y := time.Unix(first, 0).In(loc).Year(); y < start {
				start = y
			}
		}
		from = time.Date(start, 1, 1, 0, 0, 0, 0, loc)
		return from, time.Date(now.In(loc).Year()+1, 1, 1, 0, 0, 0, 0, loc), true
	}
	return from, to, false
}

// KindOf says whether a session was at home, at work or in public. where is the session's own home or public mark and
// place is the name of the named place it was at, if any: a place called Work or Office is work.
func KindOf(where, place string) string {
	p := strings.ToLower(place)
	if strings.Contains(p, "work") || strings.Contains(p, "office") {
		return "work"
	}
	if where == "home" {
		return "home"
	}
	return "public"
}

// Build fills one bar per day, month or year of from..to.
func Build(period string, from, to time.Time, trips []Trip, charges []Charge) Report {
	loc := from.Location()
	rep := Report{Period: period, From: from.Unix(), To: to.Unix(), Buckets: []Bucket{}, Kinds: []Kind{}}
	step := func(t time.Time) time.Time {
		switch period {
		case "day":
			return t.AddDate(0, 0, 1)
		case "month":
			return t.AddDate(0, 1, 0)
		}
		return t.AddDate(1, 0, 0)
	}
	layout := map[string]string{"day": "2", "month": "Jan", "year": "2006"}[period]
	for t := from; t.Before(to); t = step(t) {
		rep.Buckets = append(rep.Buckets, Bucket{Label: t.Format(layout), Start: t.Unix()})
	}
	at := func(ts int64) *Bucket {
		if ts < from.Unix() || ts >= to.Unix() {
			return nil
		}
		t := time.Unix(ts, 0).In(loc)
		var i int
		switch period {
		case "day":
			i = t.Day() - 1
		case "month":
			i = int(t.Month()) - 1
		default:
			i = t.Year() - from.Year()
		}
		if i < 0 || i >= len(rep.Buckets) {
			return nil
		}
		return &rep.Buckets[i]
	}
	for _, x := range trips {
		if b := at(x.Start); b != nil {
			b.Km += x.DistanceKm
			b.DrivingSec += x.Seconds
		}
	}
	kinds := map[string]*Kind{}
	for _, c := range charges {
		b := at(c.Start)
		if b == nil {
			continue
		}
		b.Kwh += c.AddedKwh
		b.Cost += c.Cost
		b.ChargingSec += c.Seconds
		k := kinds[c.Kind]
		if k == nil {
			k = &Kind{Kind: c.Kind}
			kinds[c.Kind] = k
		}
		k.Kwh += c.AddedKwh
		k.Cost += c.Cost
		k.Sessions++
	}
	for _, name := range []string{"home", "work", "public"} {
		if k := kinds[name]; k != nil {
			rep.Kinds = append(rep.Kinds, *k)
		}
	}
	return rep
}
