package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/ZachCurry13/cartrak/internal/analytics"
	"github.com/ZachCurry13/cartrak/internal/places"
	"github.com/ZachCurry13/cartrak/internal/store"
	"github.com/ZachCurry13/cartrak/internal/trips"
)

// The Insights pages: GET /api/analytics/{charging,driving,records,recap,battery}. ?days=90, 365 or 0 (everything) chooses
// the window; the recap takes ?month=2026-10 instead. Everyone who can see the car can see these.

func analyticsFrom(r *http.Request) int64 {
	days, err := strconv.Atoi(r.URL.Query().Get("days"))
	if err != nil || days < 0 {
		days = 365
	}
	if days == 0 {
		return 0
	}
	return time.Now().AddDate(0, 0, -days).Unix()
}

// analyticsCharges is the finished charging sessions since from, with their places named.
func (s *Server) analyticsCharges(from int64) ([]analytics.Charge, error) {
	sessions, err := s.chargeSessions(5000)
	if err != nil {
		return nil, err
	}
	named, _ := s.S.ListNamedPlaces()
	var out []analytics.Charge
	for _, c := range sessions {
		if c.Open || c.EndTS == nil || c.StartTS < from || c.AddedKwh == nil {
			continue
		}
		a := analytics.Charge{ID: c.ID, Start: c.StartTS, End: *c.EndTS, Kwh: *c.AddedKwh, Cost: c.Cost, Receipt: c.CostActual, Where: c.Where, Place: c.Place, PeakKw: c.PeakKw}
		if c.WallKwh != nil {
			a.WallKwh = *c.WallKwh
		}
		if c.Measured {
			a.Eff = c.Efficiency
		}
		if a.Place == "" && c.Where != "home" {
			a.Place = places.Name(c.Lat, c.Lon, named)
		}
		out = append(out, a)
	}
	return out, nil
}

// analyticsTrips is the finished trips since from, with the energy they used and the outside temperature.
func (s *Server) analyticsTrips(from int64) ([]analytics.Trip, error) {
	ts, err := s.S.ListTrips(store.TripFilter{From: from, All: true})
	if err != nil {
		return nil, err
	}
	temps, _ := s.S.TripTemps(from)
	capacity := s.capacityKwh()
	var done []tripOut
	for _, t := range ts {
		if !t.Open && t.EndTS != nil {
			done = append(done, tripOut{Trip: t})
		}
	}
	s.withFuel(done) // a gas or hybrid car's gallons, from the tank gauge
	var out []analytics.Trip
	for _, o := range done {
		t := o.Trip
		a := analytics.Trip{ID: t.ID, Start: t.StartTS, End: *t.EndTS, Km: t.DistanceKm, MaxKph: t.MaxSpeedKph, Category: t.Category, Corrected: t.DistanceSrc != ""}
		if e, ok := trips.EnergyUsed(t, capacity); ok {
			a.Kwh = e.Kwh
		}
		if o.FuelGal != nil {
			a.Gal = *o.FuelGal
		}
		if v, ok := temps[t.ID]; ok {
			v := v
			a.TempC = &v
		}
		out = append(out, a)
	}
	return out, nil
}

func (s *Server) analyticsCharging(w http.ResponseWriter, r *http.Request) {
	cs, err := s.analyticsCharges(analyticsFrom(r))
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, struct {
		analytics.ChargingReport
		Currency string `json:"currency"`
	}{analytics.Charging(cs, s.defaultLoc()), s.currency()})
}

func (s *Server) analyticsDriving(w http.ResponseWriter, r *http.Request) {
	ts, err := s.analyticsTrips(analyticsFrom(r))
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, analytics.Driving(ts, s.defaultLoc()))
}

func (s *Server) analyticsRecords(w http.ResponseWriter, r *http.Request) {
	ts, err1 := s.analyticsTrips(0)
	cs, err2 := s.analyticsCharges(0)
	if err1 != nil || err2 != nil {
		fail(w, 500, "db error")
		return
	}
	fills, _ := s.S.ListRefuels()
	ok(w, map[string]any{"records": append(analytics.Records(ts, cs, s.defaultLoc()), analytics.FuelRecords(ts, fills)...), "currency": s.currency()})
}

func (s *Server) analyticsRecap(w http.ResponseWriter, r *http.Request) {
	loc := s.defaultLoc()
	month := r.URL.Query().Get("month")
	if _, err := time.ParseInLocation("2006-01", month, loc); err != nil {
		month = time.Now().In(loc).Format("2006-01")
	}
	start, _ := time.ParseInLocation("2006-01", month, loc)
	from := start.AddDate(0, -1, 0).Unix() // the month before too, to compare
	ts, err1 := s.analyticsTrips(from)
	cs, err2 := s.analyticsCharges(from)
	if err1 != nil || err2 != nil {
		fail(w, 500, "db error")
		return
	}
	var gas analytics.Gas
	if s.Cfg != nil {
		g := s.Cfg.Gas()
		gas = analytics.Gas{LitersPerKm: g.LitersPerKm, PricePerLiter: g.PricePerLiter, GridKgPerKwh: s.Cfg.GridCO2()}
	}
	fills, _ := s.S.ListRefuels()
	recap := analytics.PeriodRecap(month, ts, cs, gas, loc)
	var fuelRecap *analytics.FuelRecap
	if f := analytics.FuelSummary(recap.From, recap.To, ts, fills); f.Any() {
		fuelRecap = &f
	}
	ok(w, struct {
		analytics.Recap
		Currency string               `json:"currency"`
		Fuel     *analytics.FuelRecap `json:"fuel,omitempty"`
	}{recap, s.currency(), fuelRecap})
}

func (s *Server) analyticsBattery(w http.ResponseWriter, r *http.Request) {
	from := analyticsFrom(r)
	raw, err := s.S.SocReadings(from)
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	rs := make([]analytics.Reading, len(raw))
	for i, x := range raw {
		rs[i] = analytics.Reading{TS: int64(x[0]), Pct: x[1]}
	}
	capacity, _ := s.S.DailyMedian("capacityKwh", from)
	twelve, _ := s.S.DailySeries("battery12v", from)
	// remote starts: how many, the energy they took and how long the car ran
	type remoteSum struct {
		Count   int     `json:"count"`
		Kwh     float64 `json:"kwh"`
		Minutes float64 `json:"minutes"`
	}
	var rsum remoteSum
	if list, err := s.S.RemoteStartsBetween(from, time.Now().Unix()); err == nil {
		for _, x := range list {
			rsum.Count++
			if u := x.UsedKwh(); u != nil {
				rsum.Kwh += *u
			}
			if x.End > x.Start {
				rsum.Minutes += float64(x.End-x.Start) / 60
			}
		}
	}
	ok(w, struct {
		analytics.BatteryReport
		Capacity []store.Day `json:"capacity"`
		Volts12  []store.Day `json:"volts12"`
		Remote   remoteSum   `json:"remote"`
	}{analytics.Battery(rs, s.defaultLoc()), capacity, twelve, rsum})
}
