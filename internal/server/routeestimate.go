package server

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/ZachCurry13/cartrak/internal/routeest"
	"github.com/ZachCurry13/cartrak/internal/store"
)

type estimateOut struct {
	Points [][2]float64 `json:"points"`
	Km     float64      `json:"km"`
}

func (s *Server) routeLookupOn() bool { return s.Cfg != nil && s.Cfg.RouteLookup() }

func (s *Server) routeClient() *routeest.Client {
	if s.RouteEst != nil {
		return s.RouteEst
	}
	url := ""
	if s.Cfg != nil {
		url = s.Cfg.RouteURL()
	}
	return &routeest.Client{BaseURL: url}
}

// hasPhoneTrack says whether a phone recorded this trip, so no estimate is needed.
func (s *Server) hasPhoneTrack(t *store.Trip) bool {
	end := time.Now().Unix()
	if t.EndTS != nil {
		end = *t.EndTS
	}
	return len(s.phoneRoute(t.StartTS, end)) >= 2
}

// estimateFor is the estimated road for a finished trip that has no phone GPS track. It uses the kept estimate when there
// is one. With fetch set, and the lookup switched on, it asks the routing service and keeps the answer; a failure is
// remembered for an hour so a page view never keeps hammering the service.
func (s *Server) estimateFor(t *store.Trip, fetch bool) *estimateOut {
	if t == nil || t.Open || t.EndTS == nil {
		return nil
	}
	if pts, km := s.S.RouteEstimate(t.ID); pts != nil {
		return &estimateOut{Points: pts, Km: km}
	}
	if !fetch || !s.routeLookupOn() || t.DistanceKm < 0.5 || s.hasPhoneTrack(t) {
		return nil
	}
	s.routeMu.Lock()
	if at, bad := s.routeFail[t.ID]; bad && time.Since(at) < time.Hour {
		s.routeMu.Unlock()
		return nil
	}
	s.routeMu.Unlock()
	ss, _ := s.S.TrackBetween(t.StartTS, *t.EndTS)
	var recorded []routeest.Point
	for _, x := range ss {
		if x.Lat != nil && x.Lon != nil {
			recorded = append(recorded, routeest.Point{*x.Lat, *x.Lon})
		}
	}
	if t.StartLat != nil && t.StartLon != nil && (len(recorded) == 0 || recorded[0] != (routeest.Point{*t.StartLat, *t.StartLon})) {
		recorded = append([]routeest.Point{{*t.StartLat, *t.StartLon}}, recorded...)
	}
	if t.EndLat != nil && t.EndLon != nil && (len(recorded) == 0 || recorded[len(recorded)-1] != (routeest.Point{*t.EndLat, *t.EndLon})) {
		recorded = append(recorded, routeest.Point{*t.EndLat, *t.EndLon})
	}
	fail := func() *estimateOut {
		s.routeMu.Lock()
		if s.routeFail == nil {
			s.routeFail = map[int64]time.Time{}
		}
		s.routeFail[t.ID] = time.Now()
		s.routeMu.Unlock()
		return nil
	}
	if len(recorded) < 2 {
		return fail()
	}
	road, km, err := s.routeClient().Route(context.Background(), routeest.Via(recorded, 8))
	if err != nil || !routeest.Plausible(km, t.DistanceKm) {
		return fail()
	}
	line := make([][2]float64, 0, 300)
	for _, p := range routeest.Thin(road, 300) {
		line = append(line, [2]float64(p))
	}
	_ = s.S.SaveRouteEstimate(t.ID, line, km)
	return &estimateOut{Points: line, Km: km}
}

// estimateRoutes lists, and on POST makes, estimates for older trips that have only the car's few positions.
// GET /api/trips/estimate-routes -> how many are waiting. POST does a batch (about 15), slowly, newest first.
func (s *Server) estimateRoutes(w http.ResponseWriter, r *http.Request) {
	have := s.S.RouteEstimateIDs()
	list, err := s.S.ListTrips(store.TripFilter{Limit: 300})
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	var pending []*store.Trip
	for _, t := range list {
		if t.Open || t.EndTS == nil || t.DistanceKm < 0.5 || have[t.ID] || s.hasPhoneTrack(t) {
			continue
		}
		pending = append(pending, t)
	}
	if r.Method == http.MethodGet || !s.routeLookupOn() {
		ok(w, map[string]any{"enabled": s.routeLookupOn(), "pending": len(pending)})
		return
	}
	limit := 15
	if n, _ := strconv.Atoi(r.URL.Query().Get("limit")); n > 0 && n <= 40 {
		limit = n
	}
	done, failed := 0, 0
	for i, t := range pending {
		if i >= limit || r.Context().Err() != nil {
			break
		}
		if i > 0 {
			time.Sleep(1100 * time.Millisecond) // the public routing server is for light use
		}
		if s.estimateFor(t, true) != nil {
			done++
		} else {
			failed++
		}
	}
	ok(w, map[string]any{"enabled": true, "done": done, "failed": failed, "pending": len(pending) - done})
}
