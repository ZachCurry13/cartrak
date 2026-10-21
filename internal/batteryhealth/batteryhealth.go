// Package batteryhealth turns the car's own battery-size estimate, recorded over time, into a plain verdict.
// The estimate moves with temperature and charge level, so only a clear, lasting drop is called out.
package batteryhealth

import (
	"sort"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

const (
	minDays    = 14  // days with readings before any verdict
	spanDays   = 60  // how far apart the start and the latest must be
	windowDays = 30  // the start and the latest are each averaged over this many days
	steadyPct  = 3.0 // within this a change is noise
	watchPct   = 8.0 // beyond this it is worth keeping an eye on
	dayS       = 86400
)

// Month is one calendar month's average of the estimate.
type Month struct {
	Month string  `json:"month"` // 2026-09
	Kwh   float64 `json:"kwh"`
	Days  int     `json:"days"`
}

// Report is what the Health page shows.
type Report struct {
	Ready     bool    `json:"ready"`
	Days      int     `json:"days"` // days with a reading
	StartKwh  float64 `json:"startKwh"`
	NowKwh    float64 `json:"nowKwh"`
	ChangePct float64 `json:"changePct"` // negative is a drop
	Verdict   string  `json:"verdict"`   // steady, gentle, watch or ""
	Months    []Month `json:"months"`
}

// Build summarizes daily averages (oldest first).
func Build(days []store.Day) Report {
	r := Report{Days: len(days), Months: []Month{}}
	idx := map[string]*Month{}
	var order []string
	for _, d := range days {
		k := time.Unix(d.TS, 0).UTC().Format("2006-01")
		m := idx[k]
		if m == nil {
			m = &Month{Month: k}
			idx[k] = m
			order = append(order, k)
		}
		m.Kwh += d.Avg
		m.Days++
	}
	sort.Strings(order)
	for _, k := range order {
		m := *idx[k]
		m.Kwh /= float64(m.Days)
		r.Months = append(r.Months, m)
	}
	if len(days) < minDays || days[len(days)-1].TS-days[0].TS < spanDays*dayS {
		return r
	}
	avg := func(ds []store.Day) float64 {
		var s float64
		for _, d := range ds {
			s += d.Avg
		}
		return s / float64(len(ds))
	}
	var head, tail []store.Day
	for _, d := range days {
		if d.TS < days[0].TS+windowDays*dayS {
			head = append(head, d)
		}
		if d.TS >= days[len(days)-1].TS-windowDays*dayS {
			tail = append(tail, d)
		}
	}
	r.Ready, r.StartKwh, r.NowKwh = true, avg(head), avg(tail)
	r.ChangePct = (r.NowKwh - r.StartKwh) / r.StartKwh * 100
	switch drop := -r.ChangePct; {
	case drop <= steadyPct:
		r.Verdict = "steady"
	case drop <= watchPct:
		r.Verdict = "gentle"
	default:
		r.Verdict = "watch"
	}
	return r
}
