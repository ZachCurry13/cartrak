package server

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/ZachCurry13/cartrak/internal/routeest"
)

// tripRoad: GET /api/trips/{id}/road. The length of a road between the positions a trip has, whatever the odometer said, so the owner
// can see whether the trip's distance looks too short and choose to use the road's. Admin only, because it asks the routing service.
func (s *Server) tripRoad(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	t, err := s.S.GetTrip(id)
	if err != nil || t == nil || t.Open || t.EndTS == nil {
		fail(w, 404, "no such finished trip")
		return
	}
	if !s.routeLookupOn() {
		fail(w, 409, "switch on the road estimate in More, Settings first")
		return
	}
	recorded := s.recordedPoints(t)
	if len(recorded) < 2 {
		fail(w, 409, "the car reported too few positions to find a road")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	_, km, err := s.routeClient().Route(ctx, routeest.Via(recorded, 8))
	if err != nil {
		fail(w, 502, "the routing service gave no road: "+err.Error())
		return
	}
	// the drive straight from where it started to where it ended, as a navigation app would plan it
	directKm := 0.0
	if len(recorded) > 2 {
		if _, dk, err := s.routeClient().Route(ctx, []routeest.Point{recorded[0], recorded[len(recorded)-1]}); err == nil {
			directKm = dk
		}
	} else {
		directKm = km
	}
	ok(w, map[string]any{"roadKm": km, "directKm": directKm, "straightKm": straightKm(t), "distanceKm": t.DistanceKm, "distanceSrc": t.DistanceSrc})
}

// setTripDistance: POST /api/trips/{id}/distance with {"use":"road"} (through the recorded positions), {"use":"direct"} (the drive
// straight from start to end), {"km":18.4} or {"use":"odometer"}. It corrects the distance of a
// finished trip that the odometer under- or over-counted, and remembers where the figure came from. Admin only.
func (s *Server) setTripDistance(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	var in struct {
		Use string   `json:"use"`
		Km  *float64 `json:"km"`
	}
	t, err := s.S.GetTrip(id)
	if decode(r, &in) != nil || err != nil || t == nil || t.Open || t.EndTS == nil {
		fail(w, 404, "no such finished trip")
		return
	}
	var km float64
	src := ""
	switch {
	case in.Use == "odometer":
		if t.StartOdo == nil || t.EndOdo == nil {
			fail(w, 409, "this trip has no odometer readings to go back to")
			return
		}
		km = *t.EndOdo - *t.StartOdo
	case in.Use == "road" || in.Use == "direct":
		if !s.routeLookupOn() {
			fail(w, 409, "switch on the road estimate in More, Settings first")
			return
		}
		recorded := s.recordedPoints(t)
		if len(recorded) < 2 {
			fail(w, 409, "the car reported too few positions to find a road")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		via := routeest.Via(recorded, 8)
		if in.Use == "direct" {
			via = []routeest.Point{recorded[0], recorded[len(recorded)-1]}
		}
		line, roadKm, err := s.routeClient().Route(ctx, via)
		if err != nil {
			fail(w, 502, "the routing service gave no road: "+err.Error())
			return
		}
		pts := make([][2]float64, 0, 300)
		for _, p := range routeest.Thin(line, 300) {
			pts = append(pts, [2]float64(p))
		}
		_ = s.S.SaveRouteEstimate(id, pts, roadKm) // so the road is drawn on the trip too
		km, src = roadKm, in.Use
	case in.Km != nil:
		if *in.Km < 0.1 || *in.Km > 1500 {
			fail(w, 400, "the distance must be between 0.1 and 1500 km")
			return
		}
		km, src = *in.Km, "manual"
	default:
		fail(w, 400, "choose the road, an odometer distance, or enter a distance")
		return
	}
	if err := s.S.SetTripDistance(id, km, src); err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, map[string]any{"distanceKm": km, "distanceSrc": src})
}
