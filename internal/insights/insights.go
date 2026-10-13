// Package insights looks for patterns in how efficiently the car has been driven: by month, by outside
// temperature, and the best and worst trips. It only uses finished trips with an energy figure.
package insights

import (
	"sort"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
	"github.com/ZachCurry13/cartrak/internal/trips"
)

const (
	minTripKm   = 5.0 // shorter trips say little about efficiency
	bestWorstOf = 3
)

// Band is a stretch of outside temperatures in degrees Celsius.
type Band struct {
	Name     string
	Lo, Hi   float64 // Lo <= temperature < Hi
	Trips    int
	Km, Kwh  float64
	KmPerKwh float64
}

// Month is one calendar month.
type Month struct {
	Month    string   `json:"month"` // 2026-09
	Trips    int      `json:"trips"`
	Km       float64  `json:"km"`
	Kwh      float64  `json:"kwh"`
	KmPerKwh float64  `json:"kmPerKwh"`
	AvgTempC *float64 `json:"avgTempC"`
}

// Trip is a trip picked out as especially good or bad.
type Trip struct {
	ID       int64    `json:"id"`
	StartTS  int64    `json:"startTs"`
	Km       float64  `json:"km"`
	KmPerKwh float64  `json:"kmPerKwh"`
	TempC    *float64 `json:"tempC"`
}

type BandOut struct {
	Name     string  `json:"name"`
	Trips    int     `json:"trips"`
	Km       float64 `json:"km"`
	KmPerKwh float64 `json:"kmPerKwh"`
}

type Report struct {
	Trips  int       `json:"trips"` // trips that were counted
	Months []Month   `json:"months"`
	Bands  []BandOut `json:"bands"`
	Best   []Trip    `json:"best"`
	Worst  []Trip    `json:"worst"`
}

var bands = []Band{{Name: "Below freezing", Lo: -100, Hi: 0}, {Name: "0 to 10 °C", Lo: 0, Hi: 10}, {Name: "10 to 20 °C", Lo: 10, Hi: 20}, {Name: "20 °C and above", Lo: 20, Hi: 100}}

// Build works out the report. temps maps a trip id to its average outside temperature when that is known.
func Build(ts []*store.Trip, capacityKwh *float64, temps map[int64]float64, loc *time.Location) Report {
	var rep Report
	months := map[string]*Month{}
	monthTemp := map[string][]float64{}
	bs := make([]Band, len(bands))
	copy(bs, bands)
	var all []Trip
	for _, t := range ts {
		if t.Open || t.EndTS == nil || t.DistanceKm < minTripKm {
			continue
		}
		e, ok := trips.EnergyUsed(t, capacityKwh)
		if !ok || e.Kwh <= 0 {
			continue
		}
		rep.Trips++
		key := time.Unix(t.StartTS, 0).In(loc).Format("2006-01")
		m := months[key]
		if m == nil {
			m = &Month{Month: key}
			months[key] = m
		}
		m.Trips++
		m.Km += t.DistanceKm
		m.Kwh += e.Kwh
		it := Trip{ID: t.ID, StartTS: t.StartTS, Km: t.DistanceKm, KmPerKwh: t.DistanceKm / e.Kwh}
		if temp, has := temps[t.ID]; has {
			it.TempC = &temp
			monthTemp[key] = append(monthTemp[key], temp)
			for i := range bs {
				if temp >= bs[i].Lo && temp < bs[i].Hi {
					bs[i].Trips++
					bs[i].Km += t.DistanceKm
					bs[i].Kwh += e.Kwh
				}
			}
		}
		all = append(all, it)
	}
	for k, m := range months {
		m.KmPerKwh = m.Km / m.Kwh
		if v := monthTemp[k]; len(v) > 0 {
			var sum float64
			for _, x := range v {
				sum += x
			}
			avg := sum / float64(len(v))
			m.AvgTempC = &avg
		}
		rep.Months = append(rep.Months, *m)
	}
	sort.Slice(rep.Months, func(i, j int) bool { return rep.Months[i].Month > rep.Months[j].Month })
	for _, b := range bs {
		if b.Trips > 0 && b.Kwh > 0 {
			rep.Bands = append(rep.Bands, BandOut{Name: b.Name, Trips: b.Trips, Km: b.Km, KmPerKwh: b.Km / b.Kwh})
		}
	}
	sort.Slice(all, func(i, j int) bool { return all[i].KmPerKwh > all[j].KmPerKwh })
	if len(all) >= 2*bestWorstOf {
		rep.Best = append(rep.Best, all[:bestWorstOf]...)
		rep.Worst = append(rep.Worst, all[len(all)-bestWorstOf:]...)
	}
	return rep
}
