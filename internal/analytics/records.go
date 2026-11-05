package analytics

import (
	"sort"
	"time"
)

// Record is a personal best. Kind says how to write Value: km, kmPerKwh, kph, tempC, kw, kwh, price, days, hours or trips.
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
	longestPlug := best("longestPlug", "Longest charging session", "hours", more)
	days := map[string]float64{}
	tripsPerDay := map[string]int{}
	ceiling := EffCeiling(ts)
	for _, t := range ts {
		if t.Km < 0.3 {
			continue
		}
		longest(t.Km, t.Start, t.ID, 0)
		if t.MaxKph > 0 {
			fastest(t.MaxKph, t.Start, t.ID, 0)
		}
		if t.Kwh > 0 && t.Km >= 16 && !t.Corrected {
			if e := t.Km / t.Kwh; e < ceiling {
				efficient(e, t.Start, t.ID, 0)
			}
		}
		if t.TempC != nil && t.Km >= 2 {
			coldest(*t.TempC, t.Start, t.ID, 0)
			hottest(*t.TempC, t.Start, t.ID, 0)
		}
		day := time.Unix(t.Start, 0).In(loc).Format("2006-01-02")
		days[day] += t.Km
		tripsPerDay[day]++
	}
	for _, c := range cs {
		if c.End > c.Start {
			longestPlug(float64(c.End-c.Start)/3600, c.Start, 0, c.ID)
		}
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
	if d, n := busiestDay(tripsPerDay); n > 1 {
		out = append(out, Record{Key: "mostTrips", Title: "Most trips in a day", Kind: "trips", Value: float64(n), Date: d})
	}
	if km, at, id := farthestBetweenCharges(ts, cs); id != 0 {
		out = append(out, Record{Key: "farthestCharge", Title: "Farthest between charges", Kind: "km", Value: km, TS: at, ChargeID: id})
	}
	if d, n := streak(days); n > 1 {
		out = append(out, Record{Key: "streak", Title: "Most days in a row with driving", Kind: "days", Value: float64(n), Date: d})
	}
	return out
}

// EffCeiling is the most a drive of 10 mi or more may get per kWh and still count as a best: 1.6 times what this car usually
// gets on such drives. One stale "energy left" reading at the end of a trip makes it look nearly free, and that must not
// become the record. With fewer than five such drives to judge by, a fixed 15 km/kWh stands.
func EffCeiling(ts []Trip) float64 {
	var effs []float64
	for _, t := range ts {
		if t.Kwh > 0 && t.Km >= 16 && !t.Corrected {
			effs = append(effs, t.Km/t.Kwh)
		}
	}
	if len(effs) < 5 {
		return 15
	}
	sort.Float64s(effs)
	return min(15, 1.6*effs[len(effs)/2])
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

// busiestDay is the day with the most trips, and how many.
func busiestDay(perDay map[string]int) (string, int) {
	best, day := 0, ""
	for d, n := range perDay {
		if n > best || (n == best && d > day) {
			best, day = n, d
		}
	}
	return day, best
}

// farthestBetweenCharges is the most driven between the end of one charging session and the start of the next, with the charge it
// ended at. It needs at least three sessions to mean anything.
func farthestBetweenCharges(ts []Trip, cs []Charge) (km float64, at, charge int64) {
	if len(cs) < 3 {
		return 0, 0, 0
	}
	sorted := append([]Charge(nil), cs...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Start < sorted[j].Start })
	for i := 1; i < len(sorted); i++ {
		var run float64
		for _, t := range ts {
			if t.Start >= sorted[i-1].End && t.Start < sorted[i].Start {
				run += t.Km
			}
		}
		if run > km {
			km, at, charge = run, sorted[i].Start, sorted[i].ID
		}
	}
	return km, at, charge
}
