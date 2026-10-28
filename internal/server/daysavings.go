package server

import (
	"time"

	"github.com/ZachCurry13/cartrak/internal/charging"
	"github.com/ZachCurry13/cartrak/internal/savings"
)

// dayGas is what the day's charging saved against the gas car, and each charge's own comparison, worked out charge by charge like the month and year figures:
// the distance the energy added would go, at the gas car's economy and the gas price where each charge happened. Nil when no gas
// car is set or nothing was charged that day.
func (s *Server) dayGas(from, to time.Time) (*savings.Result, map[int64]chargeGas) {
	if s.Cfg == nil || !s.Cfg.Gas().Set() {
		return nil, nil
	}
	all, _, err := s.chargeSessionsEff(1000)
	if err != nil {
		return nil, nil
	}
	var day []charging.Session
	for _, c := range all {
		if !c.Open && c.AddedKwh != nil && *c.AddedKwh > 0 && c.StartTS >= from.Unix() && c.StartTS < to.Unix() {
			day = append(day, c)
		}
	}
	if len(day) == 0 {
		return nil, nil
	}
	eq := s.gasEquivalents(day)
	r := s.savingsFor(day, eq, from, to)
	if r.GasCost == nil {
		return nil, nil
	}
	return &r, eq
}
