package server

import (
	"fmt"
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
	s.S.LogDecision("trip_deleted", fmt.Sprintf("trip %d deleted by hand (%.1f km on %s)", id, t.DistanceKm, time.Unix(t.StartTS, 0).UTC().Format("2006-01-02 15:04 UTC")))
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
	s.S.LogDecision("trip_closed", fmt.Sprintf("trip %d that never ended was closed by hand", id))
	ok(w, map[string]any{"closed": id})
}

// emptyTrips counts (GET) or deletes (POST, with {"confirm":"DELETE"}) the finished trips that went nowhere: under
// 0.2 km and never faster than 8 km/h, with no note and no journey. Admin only.
func (s *Server) emptyTrips(w http.ResponseWriter, r *http.Request) {
	ids, err := s.S.EmptyTrips()
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	if r.Method == http.MethodGet {
		ok(w, map[string]any{"count": len(ids)})
		return
	}
	var in struct {
		Confirm string `json:"confirm"`
	}
	if decode(r, &in) != nil || in.Confirm != "DELETE" {
		fail(w, 400, "confirm to delete")
		return
	}
	for _, id := range ids {
		if err := s.S.DeleteTripAndShares(id); err != nil {
			fail(w, 500, "db error")
			return
		}
	}
	ok(w, map[string]any{"deleted": len(ids)})
}
