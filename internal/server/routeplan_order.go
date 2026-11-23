package server

import (
	"context"

	"github.com/ZachCurry13/cartrak/internal/routeest"
	"github.com/ZachCurry13/cartrak/internal/trips"
)

// drivingTimes is the minutes between every pair of stops: from the road service when the road lookup is on, otherwise a straight
// line made longer. The second answer says it is the straight-line guess.
func (s *Server) drivingTimes(ctx context.Context, roads bool, stops []stopPoint) ([][]float64, bool) {
	if roads {
		pts := make([]routeest.Point, len(stops))
		for i, p := range stops {
			pts[i] = routeest.Point{p.Lat, p.Lon}
		}
		if min, _, err := s.routeClient().Table(ctx, pts); err == nil {
			return min, false
		}
	}
	n := len(stops)
	t := make([][]float64, n)
	for i := range t {
		t[i] = make([]float64, n)
		for j := range t[i] {
			if i != j {
				km := trips.Haversine(stops[i].Lat, stops[i].Lon, stops[j].Lat, stops[j].Lon) / 1000 * straightLineBend
				t[i][j] = km / straightLineKph * 60
			}
		}
	}
	return t, true
}
