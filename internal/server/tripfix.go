package server

import (
	"net/http"
	"strconv"
	"time"
)

// stuckAfter is how long an open trip must have gone without a reading before it counts as never closed.
const stuckAfter = 30 * time.Minute

// deleteTrip removes a finished trip (and any public link to it). Admin only. A trip still in progress must be
// closed first, so the detector never loses the trip it is following.
func (s *Server) deleteTrip(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	t, err := s.S.GetTrip(id)
	if err != nil || t == nil {
		fail(w, 404, "no such trip")
		return
	}
	if t.Open {
		fail(w, 409, "that trip is still open; close it first")
		return
	}
	if err := s.S.DeleteTripAndShares(id); err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, map[string]any{"deleted": id})
}

// closeTrip finishes a trip that never ended. It refuses while readings are still arriving for it.
func (s *Server) closeTrip(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	t, err := s.S.GetTrip(id)
	if err != nil || t == nil {
		fail(w, 404, "no such trip")
		return
	}
	last := t.StartTS
	if t.EndTS != nil {
		last = *t.EndTS
	}
	if !t.Open {
		fail(w, 409, "that trip is already finished")
		return
	}
	if time.Since(time.Unix(last, 0)) < stuckAfter {
		fail(w, 409, "that trip is still in progress")
		return
	}
	if _, err := s.S.CloseTrip(id); err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, map[string]any{"closed": id})
}
