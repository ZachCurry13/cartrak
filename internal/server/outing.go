package server

import (
	"fmt"
	"net/http"
	"time"
)

// outingView: GET /api/outing. What the owner set for the outing in progress ("" when nothing) and how many trips it has so far.
func (s *Server) outingView(w http.ResponseWriter, r *http.Request) {
	cat, n := s.D.Outing(time.Now().Unix())
	ok(w, map[string]any{"category": cat, "trips": n})
}

// setOuting: POST /api/outing {"category": "work"}. Sets the category of everything from leaving Home until the car is back, the
// trips so far by hand and the ones to come as they are recorded. An empty category goes back to the automatic rules.
func (s *Server) setOuting(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Category string `json:"category"`
	}
	if decode(r, &in) != nil {
		fail(w, 400, "bad request")
		return
	}
	n, err := s.D.SetOuting(in.Category, time.Now().Unix())
	if err != nil {
		fail(w, 400, err.Error())
		return
	}
	s.S.LogDecision("outing", fmt.Sprintf("this outing set to %q by hand: %d trips so far relabeled", in.Category, n))
	ok(w, map[string]any{"category": in.Category, "relabeled": n})
}
