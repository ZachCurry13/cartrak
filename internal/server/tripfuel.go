package server

import (
	"net/http"
	"sort"
	"time"

	"github.com/ZachCurry13/cartrak/internal/analytics"
	"github.com/ZachCurry13/cartrak/internal/fuel"
	"github.com/ZachCurry13/cartrak/internal/store"
)

// A gas or hybrid car's trips carry the tank level at each end, and, once the tank size has been learned from fill-ups, the
// fuel the drive burned and what it cost at the price of the latest fill-ups. All of it comes from the readings already kept, so
// older trips get it too.

const (
	startFreshSec = 3600 // the level at the start may be up to this old: a parked car reports every ten minutes or so
	endFreshSec   = 1800
)

// levelAtOrBefore is the last reported level not after ts and no older than maxAge seconds, and when it was reported.
func levelAtOrBefore(pts []store.FuelPoint, ts, maxAge int64) (pct float64, at int64, ok bool) {
	i := sort.Search(len(pts), func(i int) bool { return pts[i].TS > ts }) - 1
	if i < 0 || ts-pts[i].TS > maxAge {
		return 0, 0, false
	}
	return pts[i].Pct, pts[i].TS, true
}

// climbed says whether the gauge rose by a fill-up's worth anywhere between two times (a stop at a pump in the middle of a trip).
func climbed(pts []store.FuelPoint, from, to int64) bool {
	prev := -1.0
	for _, p := range pts {
		if p.TS < from || p.TS > to {
			continue
		}
		if prev >= 0 && p.Pct-prev >= fuel.MinRise {
			return true
		}
		prev = p.Pct
	}
	return false
}

// recentFuelPrice is what a gallon cost lately: the gallon-weighted price of the fill-ups of the last 120 days, else the latest.
func recentFuelPrice(fills []fuel.Fill) *float64 {
	cutoff := time.Now().AddDate(0, 0, -120).Unix()
	var cost, gal float64
	var last *float64
	for _, f := range fills {
		if f.Gallons == nil || f.PricePerGal == nil || *f.Gallons <= 0 {
			continue
		}
		last = f.PricePerGal
		if f.TS >= cutoff {
			cost += *f.Gallons * *f.PricePerGal
			gal += *f.Gallons
		}
	}
	if gal > 0 {
		p := cost / gal
		return &p
	}
	return last
}

// withFuel adds the tank levels, fuel burned and cost to the finished trips of a gas or hybrid car.
func (s *Server) withFuel(out []tripOut) {
	var from, to int64
	for _, o := range out {
		if o.Open || o.EndTS == nil {
			continue
		}
		if from == 0 || o.StartTS < from {
			from = o.StartTS
		}
		if *o.EndTS > to {
			to = *o.EndTS
		}
	}
	if to == 0 {
		return
	}
	pts, err := s.S.FuelBetween(from-startFreshSec, to+60)
	if err != nil || len(pts) == 0 {
		return
	}
	fills, _ := s.S.ListRefuels()
	tank, _ := fuel.TankGallons(fills)
	price := recentFuelPrice(fills)
	for i := range out {
		o := &out[i]
		if o.Open || o.EndTS == nil {
			continue
		}
		start, startAt, ok1 := levelAtOrBefore(pts, o.StartTS+60, startFreshSec)
		end, _, ok2 := levelAtOrBefore(pts, *o.EndTS+120, endFreshSec)
		if !ok1 || !ok2 || climbed(pts, startAt, *o.EndTS+120) {
			continue
		}
		o.StartFuel, o.EndFuel = &start, &end
		if k, ok := fuel.KmPerTank(o.DistanceKm, start, end); ok {
			o.KmPerTank = &k
		}
		if g, ok := fuel.Used(tank, start, end); ok {
			o.FuelGal = &g
			if price != nil {
				c := g * *price
				o.EstCost, o.Currency = &c, s.currency()
			}
		}
	}
}

// withFuelOne is withFuel for a single trip.
func (s *Server) withFuelOne(o tripOut) tripOut {
	one := []tripOut{o}
	s.withFuel(one)
	return one[0]
}

// fuelOrNil is the fuel summary of a stretch, or nil when the car burned and bought none (an electric car).
func fuelOrNil(from, to int64, ts []analytics.Trip, fills []fuel.Fill) *analytics.FuelRecap {
	if f := analytics.FuelSummary(from, to, ts, fills); f.Any() {
		return &f
	}
	return nil
}

// fuelStats: GET /api/analytics/fuel?days=365. Fuel burned and spent, in total and by month.
func (s *Server) fuelStats(w http.ResponseWriter, r *http.Request) {
	ts, err := s.analyticsTrips(0)
	fills, err2 := s.S.ListRefuels()
	if err != nil || err2 != nil {
		fail(w, 500, "db error")
		return
	}
	from := analyticsFrom(r)
	tank, n := fuel.TankGallons(fills)
	months := analytics.FuelMonths(ts, fills, s.defaultLoc())
	if from > 0 {
		keep := months[:0]
		cut := time.Unix(from, 0).In(s.defaultLoc()).Format("2006-01")
		for _, m := range months {
			if m.Period >= cut {
				keep = append(keep, m)
			}
		}
		months = keep
	}
	ok(w, map[string]any{"total": analytics.FuelSummary(from, time.Now().Unix()+1, ts, fills), "months": months, "tankGal": tank, "tankFills": n, "currency": s.currency()})
}
