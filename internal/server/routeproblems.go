package server

import "net/http"

// routeProblems: GET /api/trips/route-problems. The trips whose road could not be worked out and that were not dismissed, with why, so
// Trips can always show them: opening one trip and coming back no longer loses the rest. Admin only.
func (s *Server) routeProblems(w http.ResponseWriter, r *http.Request) {
	_ = s.S.SeedRouteReview()
	list, err := s.S.RouteReviews()
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	type item struct {
		ID      int64   `json:"id"`
		StartTS int64   `json:"startTs"`
		Why     string  `json:"why"`
		RoadKm  float64 `json:"roadKm"`
		OdoKm   float64 `json:"odoKm"`
	}
	out := []item{}
	for _, rv := range list {
		t, err := s.S.GetTrip(rv.TripID)
		if err != nil || t == nil || t.Open {
			continue
		}
		f := routeFailure{Msg: rv.Msg, RoadKm: rv.RoadKm, OdoKm: rv.OdoKm}
		why := f.text(metricReq(r))
		if why == "" {
			why = "no road was worked out for it. Open it and choose the road"
		}
		out = append(out, item{t.ID, t.StartTS, why, rv.RoadKm, rv.OdoKm})
	}
	ok(w, map[string]any{"items": out})
}
