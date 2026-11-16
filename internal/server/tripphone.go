package server

import (
	"github.com/ZachCurry13/cartrak/internal/store"
	"github.com/ZachCurry13/cartrak/internal/trips"
)

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

// phoneStepKm is the widest gap between two phone fixes that is still a track: some phone apps report only every few minutes,
// a couple of kilometers apart at highway speed. A phone put to sleep with the screen off leaves a few fixes many kilometers
// apart, and joining those with straight lines draws a road nobody drove.
const phoneStepKm = 3.0

// phoneFull says whether the phone's fixes trace the trip: at least two, none further apart than phoneStepKm. A thinner set is
// kept as places the trip passed (see recordedPoints), and its road is estimated.
func phoneFull(ps []store.Point) bool {
	if len(ps) < 2 {
		return false
	}
	for i := 1; i < len(ps); i++ {
		if trips.Haversine(ps[i-1].Lat, ps[i-1].Lon, ps[i].Lat, ps[i].Lon) > phoneStepKm*1000 {
			return false
		}
	}
	return true
}
