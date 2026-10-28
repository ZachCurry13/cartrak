package server

import (
	"time"

	"github.com/ZachCurry13/cartrak/internal/charging"
	"github.com/ZachCurry13/cartrak/internal/savings"
)

// daySavings is what the day's charging saved against the gas car, worked out charge by charge like the month and year figures:
// the distance the energy added would go, at the gas car's economy and the gas price where each charge happened. Nil when no gas
// car is set or nothing was charged that day.
func (s *Server) daySavings(from, to time.Time) *savings.Result {
	if s.Cfg == nil || !s.Cfg.Gas().Set() {
		return nil
	}
	all, _, err := s.chargeSessionsEff(1000)
	if err != nil {
		return nil
	}
	var day []charging.Session
	for _, c := range all {
		if !c.Open && c.AddedKwh != nil && *c.AddedKwh > 0 && c.StartTS >= from.Unix() && c.StartTS < to.Unix() {
			day = append(day, c)
		}
	}
	if len(day) == 0 {
		return nil
	}
	r := s.savingsFor(day, s.gasEquivalents(day), from, to)
	if r.GasCost == nil {
		return nil
	}
	return &r
}
