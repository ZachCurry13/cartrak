package analytics

import (
	"sort"
	"time"
)

// Reading is the battery level at a moment.
type Reading struct {
	TS  int64
	Pct float64
}

// SocDay is the battery's range on a day: how low and how high it went.
type SocDay struct {
	Date string  `json:"date"`
	Min  float64 `json:"min"`
	Max  float64 `json:"max"`
	End  float64 `json:"end"` // the last reading of the day
}

// SocBand is the share of time the battery spent in a range of levels.
type SocBand struct {
	From  float64 `json:"from"`
	To    float64 `json:"to"`
	Share float64 `json:"share"` // 0 to 1 of the time covered
}

// BatteryReport is the Battery story page of Insights (the capacity and 12 V trends come from their own series).
type BatteryReport struct {
	Days      []SocDay  `json:"days"`
	Bands     []SocBand `json:"bands"`
	Hours     float64   `json:"hours"`     // time the bands cover
	AvgPct    float64   `json:"avgPct"`    // the time-weighted average level
	HighShare float64   `json:"highShare"` // time at 90% or more
	LowShare  float64   `json:"lowShare"`  // time under 20%
}

var socEdges = []float64{0, 20, 50, 80, 90, 101}

// maxGap is the longest stretch one reading speaks for: the car sleeps for hours with its level unchanged, but a gap of days
// (CarTrak off) should not count.
const maxGap = 6 * 3600

// Battery builds the report from readings, oldest first.
func Battery(rs []Reading, loc *time.Location) BatteryReport {
	var r BatteryReport
	sort.Slice(rs, func(i, j int) bool { return rs[i].TS < rs[j].TS })
	secs := make([]float64, len(socEdges)-1)
	var total, weighted float64
	days := map[string]*SocDay{}
	var order []string
	for i, x := range rs {
		d := time.Unix(x.TS, 0).In(loc).Format("2006-01-02")
		day := days[d]
		if day == nil {
			day = &SocDay{Date: d, Min: x.Pct, Max: x.Pct}
			days[d] = day
			order = append(order, d)
		}
		day.Min, day.Max, day.End = min(day.Min, x.Pct), max(day.Max, x.Pct), x.Pct
		if i+1 >= len(rs) {
			continue
		}
		gap := float64(rs[i+1].TS - x.TS)
		if gap <= 0 {
			continue
		}
		gap = min(gap, maxGap)
		for b := 0; b < len(secs); b++ {
			if x.Pct >= socEdges[b] && x.Pct < socEdges[b+1] {
				secs[b] += gap
				break
			}
		}
		total += gap
		weighted += gap * x.Pct
	}
	for _, d := range order {
		r.Days = append(r.Days, *days[d])
	}
	if total <= 0 {
		return r
	}
	r.Hours = total / 3600
	r.AvgPct = weighted / total
	for b, s := range secs {
		r.Bands = append(r.Bands, SocBand{From: socEdges[b], To: min(socEdges[b+1], 100), Share: s / total})
	}
	r.LowShare = secs[0] / total
	r.HighShare = secs[len(secs)-1] / total
	return r
}
