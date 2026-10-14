// Package yearreport adds up a year of trips into the figures for a yearly summary.
package yearreport

import (
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
	"github.com/ZachCurry13/cartrak/internal/trips"
)

// Month is one calendar month of driving.
type Month struct {
	Month string  `json:"month"` // 2026-09
	Km    float64 `json:"km"`
	Trips int     `json:"trips"`
}

// Longest is the longest single trip of the year.
type Longest struct {
	ID int64   `json:"id"`
	Km float64 `json:"km"`
	TS int64   `json:"ts"`
}

// Report is the driving half of the yearly summary.
type Report struct {
	Trips       int      `json:"trips"`
	Km          float64  `json:"km"`
	WorkKm      float64  `json:"workKm"`
	PersonalKm  float64  `json:"personalKm"`
	EnergyKwh   float64  `json:"energyKwh"` // of the trips whose energy could be worked out
	EnergyKm    float64  `json:"energyKm"`  // the distance those trips covered
	EnergyTrips int      `json:"energyTrips"`
	Estimated   bool     `json:"estimated"` // some of the energy comes from battery percentage
	Longest     *Longest `json:"longest"`
	Months      []Month  `json:"months"` // every month from the first with a trip, oldest first
	BusiestKm   string   `json:"busiest"`
}

// Build totals the trips. capacityKwh may be nil.
func Build(ts []*store.Trip, capacityKwh *float64, loc *time.Location) Report {
	var r Report
	idx := map[string]*Month{}
	var order []string
	for _, t := range ts {
		r.Trips++
		r.Km += t.DistanceKm
		if t.Category == "work" {
			r.WorkKm += t.DistanceKm
		} else {
			r.PersonalKm += t.DistanceKm
		}
		if r.Longest == nil || t.DistanceKm > r.Longest.Km {
			r.Longest = &Longest{ID: t.ID, Km: t.DistanceKm, TS: t.StartTS}
		}
		if e, ok := trips.EnergyUsed(t, capacityKwh); ok && t.DistanceKm >= 1.6 {
			r.EnergyKwh += e.Kwh
			r.EnergyKm += t.DistanceKm
			r.EnergyTrips++
			r.Estimated = r.Estimated || e.Estimated
		}
		k := time.Unix(t.StartTS, 0).In(loc).Format("2006-01")
		m := idx[k]
		if m == nil {
			m = &Month{Month: k}
			idx[k] = m
			order = append(order, k)
		}
		m.Km += t.DistanceKm
		m.Trips++
	}
	best := -1.0
	for _, k := range order {
		r.Months = append(r.Months, *idx[k])
		if idx[k].Km > best {
			best, r.BusiestKm = idx[k].Km, k
		}
	}
	return r
}
