// Package journey adds up a group of trips into one: distance, driving and elapsed time, the stops between
// legs, energy used, and what the charging cost.
package journey

import (
	"github.com/ZachCurry13/cartrak/internal/charging"
	"github.com/ZachCurry13/cartrak/internal/store"
	"github.com/ZachCurry13/cartrak/internal/trips"
)

const (
	acLoss    = 0.9
	chargeSlk = 15 * 60 // a charge that began this long before a stop, or ended after it, belongs to the stop
)

// Stop is the time parked between two legs.
type Stop struct {
	FromTS   int64    `json:"fromTs"`
	ToTS     int64    `json:"toTs"`
	Seconds  int64    `json:"seconds"`
	Charging bool     `json:"charging"`
	AddedKwh *float64 `json:"addedKwh"`
	Lat      *float64 `json:"lat"`
	Lon      *float64 `json:"lon"`
}

// Summary is the whole journey in numbers.
type Summary struct {
	Trips        int      `json:"trips"`
	DistanceKm   float64  `json:"distanceKm"`
	StartTS      int64    `json:"startTs"`
	EndTS        int64    `json:"endTs"`
	MovingSec    int64    `json:"movingSec"`
	ElapsedSec   int64    `json:"elapsedSec"`
	KwhUsed      *float64 `json:"kwhUsed"`
	KwhEstimated bool     `json:"kwhEstimated"` // some of it came from battery percent
	KmPerKwh     *float64 `json:"kmPerKwh"`
	ChargedKwh   float64  `json:"chargedKwh"` // added to the battery during the stops
	Stops        []Stop   `json:"stops"`
	Cost         *float64 `json:"cost"`
	CostEstimate bool     `json:"costEstimate"` // from energy used and a price, not from priced charging sessions
}

// Summarize takes the journey's finished trips in time order, the battery capacity (may be nil), the
// charging sessions that took place in the same period, and the price per kWh to use when no session has a
// cost (may be nil).
func Summarize(ts []*store.Trip, capacityKwh *float64, sessions []charging.Session, price *float64) Summary {
	s := Summary{Stops: []Stop{}}
	var used float64
	energyTrips := 0
	for i, t := range ts {
		if t.Open || t.EndTS == nil {
			continue
		}
		s.Trips++
		s.DistanceKm += t.DistanceKm
		s.MovingSec += *t.EndTS - t.StartTS
		if s.StartTS == 0 || t.StartTS < s.StartTS {
			s.StartTS = t.StartTS
		}
		if *t.EndTS > s.EndTS {
			s.EndTS = *t.EndTS
		}
		if e, ok := trips.EnergyUsed(t, capacityKwh); ok {
			used += e.Kwh
			energyTrips++
			s.KwhEstimated = s.KwhEstimated || e.Estimated
		}
		if i > 0 && ts[i-1].EndTS != nil && t.StartTS > *ts[i-1].EndTS {
			st := Stop{FromTS: *ts[i-1].EndTS, ToTS: t.StartTS, Lat: ts[i-1].EndLat, Lon: ts[i-1].EndLon}
			st.Seconds = st.ToTS - st.FromTS
			var added float64
			for _, c := range sessions {
				if c.StartTS >= st.FromTS-chargeSlk && c.StartTS <= st.ToTS && c.AddedKwh != nil {
					st.Charging = true
					added += *c.AddedKwh
				}
			}
			if st.Charging {
				st.AddedKwh = &added
				s.ChargedKwh += added
			}
			s.Stops = append(s.Stops, st)
		}
	}
	s.ElapsedSec = s.EndTS - s.StartTS
	if energyTrips > 0 {
		s.KwhUsed = &used
		if used > 0 {
			k := s.DistanceKm / used
			s.KmPerKwh = &k
		}
	}
	var cost float64
	priced := false
	for _, c := range sessions {
		if c.Cost != nil && c.StartTS >= s.StartTS-chargeSlk && c.StartTS <= s.EndTS+chargeSlk {
			cost += *c.Cost
			priced = true
		}
	}
	switch {
	case priced:
		s.Cost = &cost
	case price != nil && s.KwhUsed != nil:
		est := *s.KwhUsed / acLoss * *price
		s.Cost, s.CostEstimate = &est, true
	}
	return s
}
