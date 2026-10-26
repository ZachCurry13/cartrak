package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

const (
	overviewTrips  = 60 // trips drawn on the overview map unless asked for fewer
	overviewMax    = 100
	overviewPoints = 40 // a trip needs no more than this to be recognizable at overview scale
)

type routeOut struct {
	ID       int64        `json:"id"`
	Category string       `json:"category"`
	StartTS  int64        `json:"startTs"`
	Points   [][2]float64 `json:"points"`
	Kind     string       `json:"kind"` // phone (GPS track), road (estimated road) or car (straight lines between the car's positions)
}

// tripRoutes gives the routes of the trips matching the same filters as the trip list, thinned
// for the overview map. A phone's GPS track is used when there is one, otherwise Ford's readings.
func (s *Server) tripRoutes(w http.ResponseWriter, r *http.Request) {
	f := tripFilter(r)
	if f.Limit <= 0 || f.Limit > overviewMax {
		f.Limit = overviewTrips
	}
	var ts []*store.Trip
	var err error
	if jid, _ := strconv.ParseInt(r.URL.Query().Get("journey"), 10, 64); jid > 0 {
		ts, err = s.S.JourneyTrips(jid) // the legs of one journey
	} else {
		ts, err = s.S.ListTrips(f)
	}
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	out := []routeOut{}
	var missing []*store.Trip
	for _, t := range ts {
		pts, kind := s.routeKind(t)
		if len(pts) >= 2 {
			out = append(out, routeOut{ID: t.ID, Category: t.Category, StartTS: t.StartTS, Points: pts, Kind: kind})
		}
		if kind == "car" && !t.Open && !s.S.RouteGaveUp(t.ID) {
			missing = append(missing, t)
		}
	}
	s.estimateLater(missing)
	ok(w, out)
}

// estimateLater asks for the roads of a few trips in the background, slowly, so the next view of the list draws them. Off
// when the road lookup is off or a batch is already running.
func (s *Server) estimateLater(ts []*store.Trip) {
	if len(ts) == 0 || !s.routeLookupOn() || !s.routeBusy.CompareAndSwap(false, true) {
		return
	}
	if len(ts) > 20 {
		ts = ts[:20] // newest first, a few a second apart: the next view of the list draws the roads
	}
	go func() {
		defer s.routeBusy.Store(false)
		for i, t := range ts {
			if i > 0 {
				time.Sleep(1100 * time.Millisecond) // the public routing server is for light use
			}
			s.estimateFor(t, true)
		}
	}()
}

func (s *Server) routeOf(t *store.Trip) [][2]float64 {
	pts, _ := s.routeKind(t)
	return pts
}

// routeKind is a trip's line and where it came from: the phone's GPS, a kept road estimate, or the car's own positions.
func (s *Server) routeKind(t *store.Trip) ([][2]float64, string) {
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
		if est := s.estimateFor(t, false); est != nil { // an estimate kept earlier beats a straight line (never asks the service here)
			return thinLine(est.Points, overviewPoints), "road"
		}
		ss, _ := s.S.TrackBetween(t.StartTS, end)
		for _, x := range ss {
			if x.Lat != nil && x.Lon != nil {
				line = append(line, [2]float64{*x.Lat, *x.Lon})
			}
		}
		return thinLine(line, overviewPoints), "car"
	}
	return thinLine(line, overviewPoints), "phone"
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
