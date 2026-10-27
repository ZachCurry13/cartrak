package server

import (
	"net/http"
	"time"

	"github.com/ZachCurry13/cartrak/internal/charging"
	"github.com/ZachCurry13/cartrak/internal/savings"
)

// savingsFor is the charging cost, cost per distance and saving against the gas car for sessions that started in [from, to).
// eq holds each charge's gas-car equivalent (see gasEquivalents); the saving is worked out charge by charge from it.
func (s *Server) savingsFor(ss []charging.Session, eq map[int64]chargeGas, from, to time.Time) savings.Result {
	in := savings.Input{GridKgPerKwh: 0.37}
	if s.Cfg != nil {
		in.GridKgPerKwh = s.Cfg.GridCO2()
	}
	var e savings.Equiv
	sources := map[string]int{}
	var wall float64
	for _, c := range ss {
		if c.Open || c.AddedKwh == nil || c.StartTS < from.Unix() || c.StartTS >= to.Unix() {
			continue
		}
		in.Sessions++
		if c.Cost != nil {
			in.Priced++
			in.ChargingCost += *c.Cost
		}
		w := *c.AddedKwh
		if c.WallKwh != nil {
			w = *c.WallKwh
		}
		in.WallKwh += w
		if eq == nil {
			continue
		}
		g, ok := eq[c.ID]
		if !ok {
			e.Skipped++
			continue
		}
		wall += w
		e.Km += g.Km
		e.Liters += g.Liters
		sources[g.Source]++
		if c.Cost != nil {
			e.GasCost += g.GasCost
			e.Paid += *c.Cost
			e.Priced++
		}
	}
	if pts, err := s.S.OdoBetween(from.Unix(), to.Unix()); err == nil && len(pts) > 1 {
		in.Km = pts[len(pts)-1].Km - pts[0].Km
	}
	if eq != nil {
		e.Sources = sourcesOf(sources)
		in.Equiv, in.WallKwh = &e, wall
	}
	return savings.Compute(in)
}

// savingsByDistance is the saving the way versions before 0.57 worked it out: the distance driven, at the owner's own gas
// price. It is kept so the page can show how the change moved the figure.
func (s *Server) savingsByDistance(ss []charging.Session, from, to time.Time) *float64 {
	if s.Cfg == nil || !s.Cfg.Gas().Set() {
		return nil
	}
	g := s.Cfg.Gas()
	in := savings.Input{LitersPerKm: g.LitersPerKm, PricePerLiter: g.PricePerLiter}
	for _, c := range ss {
		if c.Open || c.AddedKwh == nil || c.StartTS < from.Unix() || c.StartTS >= to.Unix() {
			continue
		}
		in.Sessions++
		if c.Cost != nil {
			in.Priced++
			in.ChargingCost += *c.Cost
		}
	}
	if pts, err := s.S.OdoBetween(from.Unix(), to.Unix()); err == nil && len(pts) > 1 {
		in.Km = pts[len(pts)-1].Km - pts[0].Km
	}
	return savings.Compute(in).Saved
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
	ss, _, _ := s.chargeSessionsEff(1000)
	eq := s.gasEquivalents(ss)
	out := map[string]any{"currency": s.currency(), "gasSet": gasSet, "live": s.liveGasOn(), "month": s.savingsFor(ss, eq, monthStart, end), "year": s.savingsFor(ss, eq, yearStart, end),
		"before": map[string]*float64{"month": s.savingsByDistance(ss, monthStart, end), "year": s.savingsByDistance(ss, yearStart, end)}}
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
