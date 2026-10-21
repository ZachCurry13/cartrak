package trips

import (
	"sort"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

// Window is a stretch of driving, measured as distance and the battery percent it used. Dividing one by
// the other gives the real kilometers per percent, which already includes the weather, the speeds and the
// car's hidden top and bottom buffer. That is why it needs no battery capacity.
type Window struct {
	Trips   int     `json:"trips"`
	Km      float64 `json:"km"`
	DropPct float64 `json:"dropPct"`
}

const (
	minBandTripKm  = 2 // shorter trips mostly measure rounding in the percentage
	minBandDropPct = 1
	minWindowDrop  = 8 // total percent a window needs before it says anything
	minWindowTrips = 2
)

// Measure adds up the finished trips that used at least a percent of battery.
func Measure(ts []*store.Trip) Window {
	var w Window
	for _, t := range ts {
		if t.Open || t.StartSoC == nil || t.EndSoC == nil || t.DistanceKm < minBandTripKm {
			continue
		}
		if drop := *t.StartSoC - *t.EndSoC; drop >= minBandDropPct {
			w.Trips++
			w.Km += t.DistanceKm
			w.DropPct += drop
		}
	}
	return w
}

// KmPerPct is how far one percent of battery went. It reports false when there is too little driving.
func (w Window) KmPerPct() (float64, bool) {
	if w.Trips < minWindowTrips || w.DropPct < minWindowDrop {
		return 0, false
	}
	return w.Km / w.DropPct, true
}

// KmBetween is the distance covered while the battery goes from high percent down to low percent.
func KmBetween(kmPerPct, low, high float64) float64 {
	if high <= low || kmPerPct <= 0 {
		return 0
	}
	return kmPerPct * (high - low)
}

// Month is one calendar month of driving, in the time zone each trip started in.
type Month struct {
	Month string `json:"month"` // 2026-09
	Window
}

// ByMonth groups trips by the month they started in, newest month first.
func ByMonth(ts []*store.Trip) []Month {
	groups := map[string][]*store.Trip{}
	for _, t := range ts {
		loc := time.UTC
		if l, err := time.LoadLocation(t.TZ); err == nil && t.TZ != "" {
			loc = l
		}
		key := time.Unix(t.StartTS, 0).In(loc).Format("2006-01")
		groups[key] = append(groups[key], t)
	}
	out := make([]Month, 0, len(groups))
	for k, g := range groups {
		out = append(out, Month{Month: k, Window: Measure(g)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Month > out[j].Month })
	return out
}
