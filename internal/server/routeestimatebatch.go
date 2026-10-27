package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

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
