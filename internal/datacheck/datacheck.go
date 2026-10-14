// Package datacheck looks for things in the recorded data that look wrong, so they can be seen and judged.
// It only reads; nothing here changes or deletes anything.
package datacheck

import (
	"fmt"
	"sort"
)

const (
	gapHours     = 24   // no odometer reading for this long
	backKm       = 0.5  // the odometer went back by more than this
	tinyKm       = 0.2  // a finished trip shorter than this is probably noise
	fastKph      = 220  // an average speed above this is not a real trip
	longTripHour = 24.0 // a trip longer than this has probably not been closed
)

// Odo is one odometer reading, in kilometres.
type Odo struct {
	TS int64
	Km float64
}

// Trip is the little of a trip the checks need.
type Trip struct {
	ID         int64
	StartTS    int64
	EndTS      int64 // 0 while open
	DistanceKm float64
	Open       bool
}

// Finding is one thing worth a look. TripID is set when it is about a trip, TS is when it happened.
type Finding struct {
	Kind   string `json:"kind"`  // gap, odo-back, overlap, tiny, fast, long
	Level  string `json:"level"` // info or warn
	TS     int64  `json:"ts"`
	TripID int64  `json:"tripId,omitempty"`
	Text   string `json:"text"`
}

// Check runs every check over odometer readings (oldest first) and trips, and returns the findings newest first.
func Check(odo []Odo, trips []Trip) []Finding {
	var out []Finding
	for i := 1; i < len(odo); i++ {
		a, b := odo[i-1], odo[i]
		if h := float64(b.TS-a.TS) / 3600; h >= gapHours {
			out = append(out, Finding{Kind: "gap", Level: "info", TS: a.TS, Text: fmt.Sprintf("No odometer reading for %s. Usually the car was asleep or CarTrak was not running.", span(b.TS-a.TS))})
		}
		if a.Km-b.Km > backKm {
			out = append(out, Finding{Kind: "odo-back", Level: "warn", TS: b.TS, Text: fmt.Sprintf("The odometer went back from %.1f to %.1f km. A wrong reading from Ford is the likely cause.", a.Km, b.Km)})
		}
	}
	ts := append([]Trip(nil), trips...)
	sort.Slice(ts, func(i, j int) bool { return ts[i].StartTS < ts[j].StartTS })
	var prev *Trip
	for i := range ts {
		t := &ts[i]
		if prev != nil && !prev.Open && t.StartTS < prev.EndTS {
			out = append(out, Finding{Kind: "overlap", Level: "warn", TS: t.StartTS, TripID: t.ID, Text: fmt.Sprintf("This trip starts before trip %d has ended. One of them is probably a duplicate.", prev.ID)})
		}
		prev = t
		if t.Open {
			continue
		}
		dur := float64(t.EndTS - t.StartTS)
		switch {
		case dur/3600 > longTripHour:
			out = append(out, Finding{Kind: "long", Level: "warn", TS: t.StartTS, TripID: t.ID, Text: fmt.Sprintf("This trip lasted %s. It was probably not closed when the car stopped.", span(int64(dur)))})
		case dur > 0 && t.DistanceKm/(dur/3600) > fastKph:
			out = append(out, Finding{Kind: "fast", Level: "warn", TS: t.StartTS, TripID: t.ID, Text: fmt.Sprintf("%.0f km in %s is faster than a car drives. The times or the distance are probably off.", t.DistanceKm, span(int64(dur)))})
		case t.DistanceKm < tinyKm:
			out = append(out, Finding{Kind: "tiny", Level: "info", TS: t.StartTS, TripID: t.ID, Text: "A finished trip with almost no distance. Probably the car was moved or woke up briefly."})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].TS > out[j].TS })
	return out
}

func span(sec int64) string {
	switch {
	case sec >= 2*86400:
		return fmt.Sprintf("%d days", sec/86400)
	case sec >= 3600:
		return fmt.Sprintf("%d h", sec/3600)
	default:
		return fmt.Sprintf("%d min", sec/60)
	}
}
