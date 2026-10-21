package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/ZachCurry13/cartrak/internal/analytics"
	"github.com/ZachCurry13/cartrak/internal/places"
	"github.com/ZachCurry13/cartrak/internal/store"
)

// allBadges is every trophy that applies to this car: the totals, and the habits (analytics.MoreBadges). Trophies about charging are
// left out for a car that cannot plug in, and about the tank for one with no engine.
func (s *Server) allBadges(ts []analytics.Trip, cs []analytics.Charge, towns map[string]int64, loc *time.Location) []analytics.Badge {
	fills, _ := s.S.ListRefuels()
	all := append(analytics.Badges(ts, cs, towns, loc), analytics.MoreBadges(ts, cs, fills, loc)...)
	all = analytics.WithEstimates(all, ts, time.Now())
	s.addWhere(all, cs)
	info := s.vehicleInfo()
	out := make([]analytics.Badge, 0, len(all))
	for _, b := range all {
		if (!info.Plug && analytics.EVOnly[b.Key]) || (!info.Engine && analytics.FuelOnly[b.Key]) {
			continue
		}
		out = append(out, b)
	}
	return out
}

// addWhere says where each earned trophy was earned: a place you named, else the town, of the trip or the charge that finished at the
// moment it was earned. Towns are only those already looked up (never asked for now); a trophy already reached when CarTrak began has none.
func (s *Server) addWhere(bs []analytics.Badge, cs []analytics.Charge) {
	need := false
	for _, b := range bs {
		need = need || (b.Earned > 0 && b.Note == "")
	}
	if !need {
		return
	}
	raw, err := s.S.ListTrips(store.TripFilter{All: true})
	if err != nil {
		return
	}
	named, _ := s.S.ListNamedPlaces()
	names := s.cityNamer()
	at := map[int64]string{}
	for _, t := range raw {
		if t.Open || t.EndTS == nil {
			continue
		}
		w := places.Name(t.EndLat, t.EndLon, named)
		if w == "" {
			w = names.name(t.EndLat, t.EndLon)
		}
		if w != "" {
			at[*t.EndTS] = w
		}
	}
	for _, c := range cs {
		switch {
		case c.Place != "":
			at[c.End] = c.Place
		case c.Where == "home":
			at[c.End] = "Home"
		}
	}
	names.lookupLater()
	for i := range bs {
		if bs[i].Earned > 0 && bs[i].Note == "" {
			bs[i].Where = at[bs[i].Earned]
		}
	}
}

// badgesState is every achievement with its progress. GET /api/analytics/badges
func (s *Server) badgesState(w http.ResponseWriter, r *http.Request) {
	ts, raw, err1 := s.tripsWithPlaces(0)
	cs, err2 := s.analyticsCharges(0)
	if err1 != nil || err2 != nil {
		fail(w, 500, "db error")
		return
	}
	bs := s.allBadges(ts, cs, s.towns(raw), s.defaultLoc())
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
