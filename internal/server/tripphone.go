package server

import "github.com/ZachCurry13/cartrak/internal/store"

const (
	routeSlack     = 60   // seconds of phone fixes kept before a trip starts and after it ends
	maxRoutePoints = 1000 // the route drawing does not need more than this
)

// phoneRoute returns the GPS fixes a paired phone sent during a trip, thinned for drawing.
func (s *Server) phoneRoute(startTS, endTS int64) []store.Point {
	ps, err := s.S.PointsBetween(startTS-routeSlack, endTS+routeSlack)
	if err != nil {
		return []store.Point{}
	}
	return thin(ps, maxRoutePoints)
}

// thin keeps at most max points, evenly spread, always including the first and last.
func thin(ps []store.Point, max int) []store.Point {
	if len(ps) <= max || max < 2 {
		return ps
	}
	out := make([]store.Point, 0, max)
	step := float64(len(ps)-1) / float64(max-1)
	for i := 0; i < max; i++ {
		out = append(out, ps[int(float64(i)*step+0.5)])
	}
	return out
}
