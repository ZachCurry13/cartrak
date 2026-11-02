package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/ZachCurry13/cartrak/internal/routeest"
	"github.com/ZachCurry13/cartrak/internal/store"
)

const routeOptsKeep = 15 * time.Minute

// routeOptCache keeps the roads worked out for a trip for a few minutes, so choosing one does not ask the routing service again.
type routeOptCache struct {
	mu sync.Mutex
	m  map[int64]routeOptEntry
}

type routeOptEntry struct {
	at   time.Time
	opts []routeest.Option
}

// roadOptions are the roads for a finished trip through the positions it has: the one closest to the odometer first (when the
// odometer length is trusted), then the quickest and the shortest. When the routing service cannot give alternatives for every
// stretch, the one road it gives for the whole is the only choice.
func (s *Server) roadOptions(ctx context.Context, t *store.Trip) ([]routeest.Option, error) {
	s.routeOpts.mu.Lock()
	if e, ok := s.routeOpts.m[t.ID]; ok && time.Since(e.at) < routeOptsKeep {
		s.routeOpts.mu.Unlock()
		return e.opts, nil
	}
	s.routeOpts.mu.Unlock()
	recorded := s.recordedPoints(t)
	if len(recorded) < 2 {
		return nil, fmt.Errorf("the car reported too few positions to route between")
	}
	via := routeest.Via(recorded, 8)
	opts, err := s.routeClient().Options(ctx, via, odoTarget(t))
	if err != nil || len(opts) == 0 {
		road, km, err2 := s.routeClient().RouteNear(ctx, via, odoTarget(t))
		if err2 != nil {
			return nil, err2
		}
		opts = []routeest.Option{{Kind: "quickest", Label: "Quickest", Km: km, Points: road}}
	}
	s.routeOpts.mu.Lock()
	if s.routeOpts.m == nil {
		s.routeOpts.m = map[int64]routeOptEntry{}
	}
	s.routeOpts.m[t.ID] = routeOptEntry{at: time.Now(), opts: opts}
	s.routeOpts.mu.Unlock()
	return opts, nil
}

type optionOut struct {
	Kind  string  `json:"kind"`
	Label string  `json:"label"`
	Km    float64 `json:"km"`
}

// tripRouteOptions: GET /api/trips/{id}/route-options. The roads that could be drawn for a trip, with their lengths next to what
// the odometer said, so the owner can see which fits and choose. Admin only, because it asks the routing service.
func (s *Server) tripRouteOptions(w http.ResponseWriter, r *http.Request) {
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
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	opts, err := s.roadOptions(ctx, t)
	if err != nil {
		fail(w, 502, "the routing service gave no road: "+err.Error())
		return
	}
	out := make([]optionOut, len(opts))
	for i, o := range opts {
		out[i] = optionOut{Kind: o.Kind, Label: o.Label, Km: o.Km}
	}
	_, current := s.S.RouteEstimate(t.ID)
	ok(w, map[string]any{"odoKm": t.DistanceKm, "odoTrusted": odoTarget(t) > 0, "options": out, "currentKm": current})
}

// setTripRoute: POST /api/trips/{id}/route-choice with {"kind":"closest"} (or quickest, shortest). Keeps that road as the trip's
// estimated route, even when it was tried before and found unusable. Admin only.
func (s *Server) setTripRoute(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	var in struct {
		Kind string `json:"kind"`
	}
	t, err := s.S.GetTrip(id)
	if decode(r, &in) != nil || err != nil || t == nil || t.Open || t.EndTS == nil {
		fail(w, 404, "no such finished trip")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	opts, err := s.roadOptions(ctx, t)
	if err != nil {
		fail(w, 502, "the routing service gave no road: "+err.Error())
		return
	}
	for _, o := range opts {
		if o.Kind != in.Kind {
			continue
		}
		line := make([][2]float64, 0, 300)
		for _, p := range routeest.Thin(o.Points, 300) {
			line = append(line, [2]float64(p))
		}
		if s.S.SaveRouteEstimate(t.ID, line, o.Km) != nil {
			fail(w, 500, "db error")
			return
		}
		s.routeMu.Lock()
		delete(s.routeFail, t.ID)
		s.routeMu.Unlock()
		ok(w, map[string]any{"km": o.Km, "kind": o.Kind})
		return
	}
	fail(w, 404, "that road is not one of the choices")
}
