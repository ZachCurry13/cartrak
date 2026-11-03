package server

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/ZachCurry13/cartrak/internal/routeest"
	"github.com/ZachCurry13/cartrak/internal/store"
	"github.com/ZachCurry13/cartrak/internal/trips"
)

const maxTripStops = 5

// withStops puts each stop into the recorded positions where it adds the least detour, so a trip the car recorded with two positions
// and a stop in between is drawn start, stop, end, and one with many positions gets the stop beside the nearest stretch.
func withStops(recorded []routeest.Point, stops []store.Via) []routeest.Point {
	out := append([]routeest.Point(nil), recorded...)
	for _, v := range stops {
		p := routeest.Point{v.Lat, v.Lon}
		if len(out) < 2 {
			out = append(out, p)
			continue
		}
		best, bestAdd := 1, 1e18
		for i := 1; i < len(out); i++ {
			a, b := out[i-1], out[i]
			add := trips.Haversine(a[0], a[1], p[0], p[1]) + trips.Haversine(p[0], p[1], b[0], b[1]) - trips.Haversine(a[0], a[1], b[0], b[1])
			if add < bestAdd {
				best, bestAdd = i, add
			}
		}
		out = append(out[:best], append([]routeest.Point{p}, out[best:]...)...)
	}
	return out
}

func cleanStops(in []store.Via) ([]store.Via, bool) {
	if len(in) > maxTripStops {
		return nil, false
	}
	out := make([]store.Via, 0, len(in))
	for _, v := range in {
		if v.Lat < -90 || v.Lat > 90 || v.Lon < -180 || v.Lon > 180 || (v.Lat == 0 && v.Lon == 0) {
			return nil, false
		}
		name := strings.TrimSpace(v.Name)
		if len(name) > 60 {
			name = name[:60]
		}
		out = append(out, store.Via{Name: name, Lat: v.Lat, Lon: v.Lon})
	}
	return out, true
}

// tripStops lists the stops added to a trip's route. GET /api/trips/{id}/route-stops
func (s *Server) tripStops(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	v, err := s.S.TripVia(id)
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, map[string]any{"stops": v, "max": maxTripStops})
}

// setTripStops replaces the stops of a trip's route and forgets the roads worked out without them; the caller asks for the roads
// again. POST /api/trips/{id}/route-stops {stops:[{name,lat,lon}]}. Admin only.
func (s *Server) setTripStops(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	var in struct{ Stops []store.Via }
	t, err := s.S.GetTrip(id)
	if decode(r, &in) != nil || err != nil || t == nil || t.Open || t.EndTS == nil {
		fail(w, 404, "no such finished trip")
		return
	}
	stops, valid := cleanStops(in.Stops)
	if !valid {
		fail(w, 400, "a trip can have up to "+strconv.Itoa(maxTripStops)+" stops, each a real place")
		return
	}
	if err := s.S.SetTripVia(id, stops); err != nil {
		fail(w, 500, "db error")
		return
	}
	s.routeOpts.mu.Lock()
	delete(s.routeOpts.m, id) // the roads offered before did not go through these stops
	s.routeOpts.mu.Unlock()
	s.routeMu.Lock()
	delete(s.routeFail, id)
	s.routeMu.Unlock()
	ok(w, map[string]any{"stops": stops})
}
