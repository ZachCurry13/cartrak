package analytics

import (
	"sort"
	"time"
)

// Record is a personal best. Kind says how to write Value: km, kmPerKwh, kph, tempC, kw, kwh, price or days.
type Record struct {
	Key      string  `json:"key"`
	Title    string  `json:"title"`
	Kind     string  `json:"kind"`
	Value    float64 `json:"value"`
	TS       int64   `json:"ts"`
	TripID   int64   `json:"tripId,omitempty"`
	ChargeID int64   `json:"chargeId,omitempty"`
	Date     string  `json:"date,omitempty"` // for a day or a streak: the (last) day
}

// Records finds the personal bests among finished trips and charges.
func Records(ts []Trip, cs []Charge, loc *time.Location) []Record {
	var out []Record
	best := func(key, title, kind string, better func(a, b float64) bool) func(v float64, ts int64, trip, charge int64) {
		i := -1
		return func(v float64, at int64, trip, charge int64) {
			if i < 0 {
				out = append(out, Record{Key: key, Title: title, Kind: kind, Value: v, TS: at, TripID: trip, ChargeID: charge})
				i = len(out) - 1
				return
			}
			if better(v, out[i].Value) {
				out[i] = Record{Key: key, Title: title, Kind: kind, Value: v, TS: at, TripID: trip, ChargeID: charge}
			}
		}
	}
	more := func(a, b float64) bool { return a > b }
	less := func(a, b float64) bool { return a < b }
	longest := best("longest", "Longest drive", "km", more)
	efficient := best("efficient", "Most efficient drive (10 mi or more)", "kmPerKwh", more)
	fastest := best("topSpeed", "Top speed", "kph", more)
	coldest := best("coldest", "Coldest drive", "tempC", less)
	hottest := best("hottest", "Hottest drive", "tempC", more)
	peak := best("peakKw", "Fastest charging", "kw", more)
	biggest := best("bigCharge", "Most energy in one charge", "kwh", more)
	cheapest := best("cheapest", "Cheapest public charge (paid, 5 kWh or more)", "price", less)
	days := map[string]float64{}
	for _, t := range ts {
		if t.Km < 0.3 {
			continue
		}
		longest(t.Km, t.Start, t.ID, 0)
		if t.MaxKph > 0 {
			fastest(t.MaxKph, t.Start, t.ID, 0)
		}
		if t.Kwh > 0 && t.Km >= 16 {
			if e := t.Km / t.Kwh; e < 15 {
				efficient(e, t.Start, t.ID, 0)
			}
		}
		if t.TempC != nil && t.Km >= 2 {
			coldest(*t.TempC, t.Start, t.ID, 0)
			hottest(*t.TempC, t.Start, t.ID, 0)
		}
		days[time.Unix(t.Start, 0).In(loc).Format("2006-01-02")] += t.Km
	}
	for _, c := range cs {
		if c.PeakKw > 0 {
			peak(c.PeakKw, c.Start, 0, c.ID)
		}
		if c.Kwh > 0 {
			biggest(c.Kwh, c.Start, 0, c.ID)
		}
		if c.Cost != nil && *c.Cost > 0 && c.Where != "home" && c.Kwh >= 5 { // home charging and free chargers would always win
			wall := c.WallKwh
			if wall <= 0 {
				wall = c.Kwh
			}
			cheapest(*c.Cost/wall, c.Start, 0, c.ID)
		}
	}
	if d, km := biggestDay(days); d != "" {
		out = append(out, Record{Key: "bigDay", Title: "Most driving in a day", Kind: "km", Value: km, Date: d})
	}
	if d, n := streak(days); n > 1 {
		out = append(out, Record{Key: "streak", Title: "Most days in a row with driving", Kind: "days", Value: float64(n), Date: d})
	}
	return out
}

func biggestDay(days map[string]float64) (string, float64) {
	best, km := "", 0.0
	for d, v := range days {
		if v > km || (v == km && d > best) {
			best, km = d, v
		}
	}
	return best, km
}

// streak is the longest run of days in a row with a drive, and the day it ended.
func streak(days map[string]float64) (string, int) {
	keys := make([]string, 0, len(days))
	for d := range days {
		keys = append(keys, d)
	}
	sort.Strings(keys)
	bestEnd, best, run := "", 0, 0
	var prev time.Time
	for _, k := range keys {
		t, err := time.Parse("2006-01-02", k)
		if err != nil {
			continue
		}
		if run > 0 && t.Sub(prev) == 24*time.Hour {
			run++
		} else {
			run = 1
		}
		prev = t
		if run >= best {
			best, bestEnd = run, k
		}
	}
	return bestEnd, best
}
