package server

import (
	"net/http"
	"time"

	"github.com/ZachCurry13/cartrak/internal/planahead"
	"github.com/ZachCurry13/cartrak/internal/savings"
	"github.com/ZachCurry13/cartrak/internal/store"
)

// savingsFor is the charging cost, cost per distance and saving against the gas car for sessions that started in [from, to).
func (s *Server) savingsFor(from, to time.Time) savings.Result {
	in := savings.Input{GridKgPerKwh: 0.37}
	if s.Cfg != nil {
		p := s.Cfg.Gas()
		in.LitresPerKm, in.PricePerLitre, in.GridKgPerKwh = p.LitresPerKm, p.PricePerLitre, s.Cfg.GridCO2()
	}
	if ss, _, err := s.chargeSessionsEff(1000); err == nil {
		for _, c := range ss {
			if c.Open || c.AddedKwh == nil || c.StartTS < from.Unix() || c.StartTS >= to.Unix() {
				continue
			}
			in.Sessions++
			if c.Cost != nil {
				in.Priced++
				in.ChargingCost += *c.Cost
			}
			if c.WallKwh != nil {
				in.WallKwh += *c.WallKwh
			} else {
				in.WallKwh += *c.AddedKwh
			}
		}
	}
	if pts, err := s.S.OdoBetween(from.Unix(), to.Unix()); err == nil && len(pts) > 1 {
		in.Km = pts[len(pts)-1].Km - pts[0].Km
	}
	return savings.Compute(in)
}

// savingsView is the month and the year so far: what charging cost per distance, and what driving electric saved against the
// gas car set in the charging settings.
func (s *Server) savingsView(w http.ResponseWriter, r *http.Request) {
	loc, now := s.defaultLoc(), time.Now()
	local := now.In(loc)
	monthStart := time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, loc)
	yearStart := time.Date(local.Year(), 1, 1, 0, 0, 0, 0, loc)
	end := now.Add(time.Hour)
	gasSet := s.Cfg != nil && s.Cfg.Gas().Set()
	out := map[string]any{"currency": s.currency(), "gasSet": gasSet, "month": s.savingsFor(monthStart, end), "year": s.savingsFor(yearStart, end)}
	ok(w, out)
}

// avgPricePerKwh is what a kWh has cost you lately: the total of the priced sessions over the energy taken, else the home price.
func (s *Server) avgPricePerKwh() *float64 {
	var cost, kwh float64
	cutoff := time.Now().AddDate(0, 0, -120).Unix()
	if ss, _, err := s.chargeSessionsEff(300); err == nil {
		for _, c := range ss {
			if c.Open || c.Cost == nil || c.WallKwh == nil || *c.WallKwh <= 0 || c.StartTS < cutoff {
				continue
			}
			cost += *c.Cost
			kwh += *c.WallKwh
		}
	}
	if kwh > 0 {
		p := cost / kwh
		return &p
	}
	return s.prices().Home
}

// withTripCost adds an estimate of what a trip's energy cost, at the average price per kWh. It is a guess and says so.
func (s *Server) withTripCost(t *tripOut) {
	if t.EnergyKwh == nil {
		return
	}
	if p := s.avgPricePerKwh(); p != nil {
		c := *t.EnergyKwh * *p
		t.EstCost, t.Currency = &c, s.currency()
	}
}

// planAhead forecasts the next week of driving from the last eight weeks and says when the battery would need a charge.
func (s *Server) planAhead(w http.ResponseWriter, r *http.Request) {
	loc := s.defaultLoc()
	now := time.Now().In(loc)
	midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	start := midnight.AddDate(0, 0, -56)
	if first := s.S.FirstSampleTS(); first > 0 {
		f := time.Unix(first, 0).In(loc)
		if fd := time.Date(f.Year(), f.Month(), f.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, 1); fd.After(start) {
			start = fd // the first day is partial, so history starts the day after
		}
	}
	x, _ := s.S.LatestSample()
	capacity := s.capacityKwh()
	if x == nil || x.SoCPct == nil || capacity == nil {
		ok(w, map[string]any{"ready": false})
		return
	}
	ts, err := s.S.ListTrips(store.TripFilter{From: start.Unix(), Limit: 5000})
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	perDay := map[string]float64{}
	for _, t := range ts {
		if !t.Open {
			perDay[time.Unix(t.StartTS, 0).In(loc).Format("2006-01-02")] += t.DistanceKm
		}
	}
	var history []planahead.DayKm
	for d := start; d.Before(midnight); d = d.AddDate(0, 0, 1) {
		k := d.Format("2006-01-02")
		history = append(history, planahead.DayKm{Date: k, Km: perDay[k]})
	}
	var kwh, km float64
	for _, t := range s.tripsWithEnergy(ts) {
		if t.EnergyKwh != nil && t.DistanceKm > 1 {
			kwh += *t.EnergyKwh
			km += t.DistanceKm
		}
	}
	reserve := 20.0
	if s.Cfg != nil {
		if l := s.Cfg.LowBatteryLevel(); l > 0 {
			reserve = l
		}
	}
	if km <= 0 {
		ok(w, map[string]any{"ready": false})
		return
	}
	f, good := planahead.Build(planahead.Input{History: history, Today: midnight, KwhPerKm: kwh / km, CapacityKwh: *capacity, StartPct: *x.SoCPct, ReservePct: reserve})
	if !good {
		ok(w, map[string]any{"ready": false, "historyDays": len(history), "needDays": planahead.MinHistoryDays})
		return
	}
	ok(w, map[string]any{"ready": true, "forecast": f, "kwhPerKm": kwh / km})
}
