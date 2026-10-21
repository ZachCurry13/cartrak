package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/ZachCurry13/cartrak/internal/analytics"
	"github.com/ZachCurry13/cartrak/internal/store"
)

// badgesState is every achievement with its progress. GET /api/analytics/badges
func (s *Server) badgesState(w http.ResponseWriter, r *http.Request) {
	ts, raw, err1 := s.tripsWithPlaces(0)
	cs, err2 := s.analyticsCharges(0)
	if err1 != nil || err2 != nil {
		fail(w, 500, "db error")
		return
	}
	bs := analytics.Badges(ts, cs, s.towns(raw), s.defaultLoc())
	type out struct {
		analytics.Badge
		Progress float64 `json:"progress"`
	}
	list := make([]out, len(bs))
	for i, b := range bs {
		list[i] = out{b, b.Progress()}
	}
	ok(w, list)
}

// everywhere is every route driven in the window, thinned, for the glowing map. GET /api/analytics/everywhere?days=365, or
// ?period=2026-10 for a month or a year.
func (s *Server) everywhere(w http.ResponseWriter, r *http.Request) {
	f := store.TripFilter{From: analyticsFrom(r), Limit: 1000}
	if from, to, _, ok := analytics.Period(r.URL.Query().Get("period"), s.defaultLoc()); ok {
		f.From, f.To = from, to
	}
	ts, err := s.S.ListTrips(f)
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	routes := [][][2]float64{}
	var km float64
	drives := 0
	for _, t := range ts {
		if t.Open || t.EndTS == nil || t.DistanceKm < 0.3 {
			continue
		}
		km += t.DistanceKm
		drives++
		pts, _ := s.routeKind(t)
		if len(pts) < 2 {
			continue
		}
		line := thinLine(pts, 30)
		for i, p := range line {
			line[i] = [2]float64{round4(p[0]), round4(p[1])}
		}
		routes = append(routes, line)
	}
	ok(w, map[string]any{"routes": routes, "trips": drives, "km": km}) // a drive with no positions has no line
}

func round4(v float64) float64 { return float64(int64(v*1e4+0.5*sign(v))) / 1e4 }

func sign(v float64) float64 {
	if v < 0 {
		return -1
	}
	return 1
}

// life is the energy flow and the typical day over the window. GET /api/analytics/life?days=30
func (s *Server) life(w http.ResponseWriter, r *http.Request) {
	days, err := strconv.Atoi(r.URL.Query().Get("days"))
	if err != nil || days <= 0 || days > 3650 {
		days = 30
	}
	now := time.Now().Unix()
	from := now - int64(days)*86400
	ts, err1 := s.analyticsTrips(from)
	cs, err2 := s.analyticsCharges(from)
	if err1 != nil || err2 != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, map[string]any{"flow": analytics.EnergyFlow(ts, cs, from, now, s.storedKwh(from, now)), "ring": analytics.TypicalDay(ts, cs, from, now, s.defaultLoc()), "days": days})
}
