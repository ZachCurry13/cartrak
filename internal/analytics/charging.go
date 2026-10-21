// Package analytics works out the Insights pages: charging, driving patterns, records and the monthly recap, and the
// battery's story. It takes plain inputs the server fills in, so each part is easy to test, and it never sees positions.
package analytics

import (
	"sort"
	"time"
)

// Charge is one finished charging session as the server sees it.
type Charge struct {
	ID      int64
	Start   int64
	End     int64
	Kwh     float64  // into the battery
	WallKwh float64  // from the charger, allowing for losses (0 when unknown)
	Cost    *float64 // nil when unpriced
	Receipt bool     // the cost is what was paid
	Where   string   // home or public
	Place   string   // a named place, or "" (then Where names it)
	PeakKw  float64
	Eff     *float64 // measured share that reached the battery, when the charger's kWh was entered
}

// ChargeMonth is one month of charging, home and public apart.
type ChargeMonth struct {
	Month       string  `json:"month"` // 2026-10
	HomeKwh     float64 `json:"homeKwh"`
	PublicKwh   float64 `json:"publicKwh"`
	HomeCost    float64 `json:"homeCost"`
	PublicCost  float64 `json:"publicCost"`
	Sessions    int     `json:"sessions"`
	PricePerKwh float64 `json:"pricePerKwh"` // what a kWh from the charger cost on average, 0 when nothing was priced
	Eff         float64 `json:"eff"`         // measured efficiency that month, 0 when none
}

// ChargePlace is where energy came from.
type ChargePlace struct {
	Name     string  `json:"name"`
	Where    string  `json:"where"`
	Sessions int     `json:"sessions"`
	Kwh      float64 `json:"kwh"`
	Cost     float64 `json:"cost"`
}

// FastCharge is one of the quickest sessions, whose curve the page can draw.
type FastCharge struct {
	ID     int64   `json:"id"`
	Start  int64   `json:"start"`
	PeakKw float64 `json:"peakKw"`
	Kwh    float64 `json:"kwh"`
	Place  string  `json:"place"`
}

// ChargingReport is the Charging page of Insights.
type ChargingReport struct {
	Sessions  int           `json:"sessions"`
	Kwh       float64       `json:"kwh"`
	Cost      float64       `json:"cost"`
	HomeShare float64       `json:"homeShare"` // share of the energy charged at home, 0 to 1
	Months    []ChargeMonth `json:"months"`    // oldest first
	Places    []ChargePlace `json:"places"`    // most energy first, at most eight
	PlugIn    [7][24]int    `json:"plugIn"`    // sessions started by weekday (Sunday first) and hour
	Fastest   []FastCharge  `json:"fastest"`   // up to five, highest peak first
}

func monthOf(ts int64, loc *time.Location) string { return time.Unix(ts, 0).In(loc).Format("2006-01") }

// Charging builds the report from finished sessions.
func Charging(cs []Charge, loc *time.Location) ChargingReport {
	var r ChargingReport
	months := map[string]*ChargeMonth{}
	paid := map[string][2]float64{} // month: cost, charger kWh of the priced sessions
	effs := map[string][2]float64{} // month: sum, count
	places := map[string]*ChargePlace{}
	var homeKwh float64
	for _, c := range cs {
		if c.Kwh <= 0 {
			continue
		}
		r.Sessions++
		r.Kwh += c.Kwh
		key := monthOf(c.Start, loc)
		m := months[key]
		if m == nil {
			m = &ChargeMonth{Month: key}
			months[key] = m
		}
		m.Sessions++
		cost := 0.0
		if c.Cost != nil {
			cost = *c.Cost
			wall := c.WallKwh
			if wall <= 0 {
				wall = c.Kwh
			}
			p := paid[key]
			paid[key] = [2]float64{p[0] + cost, p[1] + wall}
		}
		r.Cost += cost
		if c.Where == "home" {
			m.HomeKwh += c.Kwh
			m.HomeCost += cost
			homeKwh += c.Kwh
		} else {
			m.PublicKwh += c.Kwh
			m.PublicCost += cost
		}
		if c.Eff != nil {
			e := effs[key]
			effs[key] = [2]float64{e[0] + *c.Eff, e[1] + 1}
		}
		name := c.Place
		if name == "" {
			name = map[string]string{"home": "Home"}[c.Where]
			if name == "" {
				name = "Other public chargers"
			}
		}
		p := places[name]
		if p == nil {
			p = &ChargePlace{Name: name, Where: c.Where}
			places[name] = p
		}
		p.Sessions++
		p.Kwh += c.Kwh
		p.Cost += cost
		t := time.Unix(c.Start, 0).In(loc)
		r.PlugIn[int(t.Weekday())][t.Hour()]++
		if c.PeakKw >= 30 {
			r.Fastest = append(r.Fastest, FastCharge{ID: c.ID, Start: c.Start, PeakKw: c.PeakKw, Kwh: c.Kwh, Place: name})
		}
	}
	if r.Kwh > 0 {
		r.HomeShare = homeKwh / r.Kwh
	}
	for k, m := range months {
		if p := paid[k]; p[1] > 0 {
			m.PricePerKwh = p[0] / p[1]
		}
		if e := effs[k]; e[1] > 0 {
			m.Eff = e[0] / e[1]
		}
		r.Months = append(r.Months, *m)
	}
	sort.Slice(r.Months, func(i, j int) bool { return r.Months[i].Month < r.Months[j].Month })
	for _, p := range places {
		r.Places = append(r.Places, *p)
	}
	sort.Slice(r.Places, func(i, j int) bool { return r.Places[i].Kwh > r.Places[j].Kwh })
	if len(r.Places) > 8 {
		r.Places = r.Places[:8]
	}
	sort.Slice(r.Fastest, func(i, j int) bool { return r.Fastest[i].PeakKw > r.Fastest[j].PeakKw })
	if len(r.Fastest) > 5 {
		r.Fastest = r.Fastest[:5]
	}
	return r
}
