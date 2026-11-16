package server

import (
	"net/http"

	"github.com/ZachCurry13/cartrak/internal/analytics"
)

// lifetime is the estimate of the car's whole life (see analytics.EstimateLifetime), for a car that charges.
func (s *Server) lifetime(ts []analytics.Trip, cs []analytics.Charge) (analytics.Lifetime, bool) {
	x, _ := s.S.LatestSample()
	if x == nil || x.OdometerKm == nil || !s.vehicleInfo().Plug {
		return analytics.Lifetime{}, false
	}
	return analytics.EstimateLifetime(ts, cs, *x.OdometerKm)
}

// lifetimeAPI: GET /api/lifetime. {"ok": false} until there is enough driving to go on.
func (s *Server) lifetimeAPI(w http.ResponseWriter, r *http.Request) {
	ts, err1 := s.analyticsTrips(0)
	cs, err2 := s.analyticsCharges(0)
	if err1 != nil || err2 != nil {
		fail(w, http.StatusInternalServerError, "could not read the trips")
		return
	}
	l, ok := s.lifetime(ts, cs)
	if !ok {
		s.ok2(w, false, nil)
		return
	}
	s.ok2(w, true, &l)
}

func (s *Server) ok2(w http.ResponseWriter, enough bool, l *analytics.Lifetime) {
	ok(w, map[string]any{"ok": enough, "lifetime": l, "currency": s.currency()})
}
