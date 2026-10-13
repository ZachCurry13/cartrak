package server

import (
	"net/http"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

const (
	overviewTrips  = 60 // trips drawn on the overview map unless asked for fewer
	overviewMax    = 100
	overviewPoints = 40 // a trip needs no more than this to be recognisable at overview scale
)

type routeOut struct {
	ID       int64        `json:"id"`
	Category string       `json:"category"`
	StartTS  int64        `json:"startTs"`
	Points   [][2]float64 `json:"points"`
}

// tripRoutes gives the routes of the trips matching the same filters as the trip list, thinned
// for the overview map. A phone's GPS track is used when there is one, otherwise Ford's readings.
func (s *Server) tripRoutes(w http.ResponseWriter, r *http.Request) {
	f := tripFilter(r)
	if f.Limit <= 0 || f.Limit > overviewMax {
		f.Limit = overviewTrips
	}
	ts, err := s.S.ListTrips(f)
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	out := []routeOut{}
	for _, t := range ts {
		if pts := s.routeOf(t); len(pts) >= 2 {
			out = append(out, routeOut{ID: t.ID, Category: t.Category, StartTS: t.StartTS, Points: pts})
		}
	}
	ok(w, out)
}

func (s *Server) routeOf(t *store.Trip) [][2]float64 {
	end := time.Now().Unix()
	if t.EndTS != nil {
		end = *t.EndTS
	}
	var line [][2]float64
	for _, p := range s.phoneRoute(t.StartTS, end) {
		line = append(line, [2]float64{p.Lat, p.Lon})
	}
	if len(line) < 2 {
		line = line[:0]
		ss, _ := s.S.TrackBetween(t.StartTS, end)
		for _, x := range ss {
			if x.Lat != nil && x.Lon != nil {
				line = append(line, [2]float64{*x.Lat, *x.Lon})
			}
		}
	}
	return thinLine(line, overviewPoints)
}

// thinLine keeps at most max points, evenly spread, always including the first and last.
func thinLine(ps [][2]float64, max int) [][2]float64 {
	if len(ps) <= max || max < 2 {
		return ps
	}
	out := make([][2]float64, 0, max)
	step := float64(len(ps)-1) / float64(max-1)
	for i := 0; i < max; i++ {
		out = append(out, ps[int(float64(i)*step+0.5)])
	}
	return out
}
