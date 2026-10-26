package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/ZachCurry13/cartrak/internal/charging"
	"github.com/ZachCurry13/cartrak/internal/routeest"
	"github.com/ZachCurry13/cartrak/internal/store"
)

type estimateOut struct {
	Points [][2]float64 `json:"points"`
	Km     float64      `json:"km"`
}

// routeFailure is why a trip's road could not be estimated. RoadKm is set when the routing service did answer, with a
// road of a different length from the odometer.
type routeFailure struct {
	At     time.Time
	Msg    string
	RoadKm float64
	OdoKm  float64
}

// text writes the reason in the units the person uses.
func (f routeFailure) text(metric bool) string {
	if f.RoadKm > 0 {
		return fmt.Sprintf("the road found was %s but the odometer says %s", distText(f.RoadKm, metric, 1), distText(f.OdoKm, metric, 1))
	}
	return f.Msg
}

// straightKm is the straight line from a trip's start to its end, 0 when either is unknown.
func straightKm(t *store.Trip) float64 {
	if t.StartLat == nil || t.StartLon == nil || t.EndLat == nil || t.EndLon == nil {
		return 0
	}
	return charging.DistanceM(charging.Point{Lat: *t.StartLat, Lon: *t.StartLon}, charging.Point{Lat: *t.EndLat, Lon: *t.EndLon}) / 1000
}

// retryGiveUpsOnce forgets, once, the trips whose road was given up on before the check also allowed for a short odometer,
// so they are asked about again under the new rule.
func (s *Server) retryGiveUpsOnce() {
	if s.routeRetried.Load() {
		return
	}
	s.routeRetried.Store(true)
	if s.S.GetKV("route_retry") != "1" {
		s.S.ForgetNoRoutes()
		_ = s.S.SetKV("route_retry", "1")
	}
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

// recordedPoints are the positions a trip has: the car's own, with its start and end added when they are missing.
func (s *Server) recordedPoints(t *store.Trip) []routeest.Point {
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
	return recorded
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
	s.retryGiveUpsOnce()
	if s.S.RouteGaveUp(t.ID) {
		return nil
	}
	if !fetch || !s.routeLookupOn() || t.DistanceKm < 0.5 || s.hasPhoneTrack(t) {
		return nil
	}
	s.routeMu.Lock()
	if f, bad := s.routeFail[t.ID]; bad && time.Since(f.At) < time.Hour {
		s.routeMu.Unlock()
		return nil
	}
	s.routeMu.Unlock()
	recorded := s.recordedPoints(t)
	fail := func(f routeFailure) *estimateOut {
		f.At = time.Now()
		s.routeMu.Lock()
		if s.routeFail == nil {
			s.routeFail = map[int64]routeFailure{}
		}
		s.routeFail[t.ID] = f
		s.routeMu.Unlock()
		return nil
	}
	if len(recorded) < 2 {
		return fail(routeFailure{Msg: "the car reported too few positions to route between"})
	}
	road, km, err := s.routeClient().RouteNear(context.Background(), routeest.Via(recorded, 8), odoTarget(t))
	if err != nil {
		return fail(routeFailure{Msg: err.Error()})
	}
	if !routeest.Plausible(km, t.DistanceKm, straightKm(t)) {
		_ = s.S.SaveNoRoute(t.ID) // asking again would give the same road
		return fail(routeFailure{RoadKm: km, OdoKm: t.DistanceKm})
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
	type failedTrip struct {
		ID      int64   `json:"id"`
		StartTS int64   `json:"startTs"`
		Why     string  `json:"why"`
		RoadKm  float64 `json:"roadKm"` // the length of the road found, when one was found: the owner can choose to use it
		OdoKm   float64 `json:"odoKm"`
	}
	var failures []failedTrip
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
			s.routeMu.Lock()
			f := s.routeFail[t.ID]
			s.routeMu.Unlock()
			failures = append(failures, failedTrip{t.ID, t.StartTS, f.text(metricReq(r)), f.RoadKm, f.OdoKm})
		}
	}
	ok(w, map[string]any{"enabled": true, "done": done, "failed": failed, "pending": len(pending) - done, "failures": failures})
}

// odoTarget is the length the road for a trip should be close to: what the odometer recorded, unless that was corrected by hand
// (then the choice is the owner's) or is too small to mean anything.
func odoTarget(t *store.Trip) float64 {
	if t.DistanceSrc != "" || t.DistanceKm < 0.5 {
		return 0
	}
	return t.DistanceKm
}
