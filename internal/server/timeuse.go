package server

import (
	"net/http"
	"time"

	"github.com/ZachCurry13/cartrak/internal/timeuse"
)

// timeUse is how the last 7 days, 30 days or year were spent: driving, charging or parked, by day and by place.
func (s *Server) timeUse(w http.ResponseWriter, r *http.Request) {
	days := intQuery(r, "days", 30, 1, 366)
	now := time.Now()
	from := now.AddDate(0, 0, -days).Unix()
	// Trips from before the window are needed to know where the car was parked when it began.
	ts, err := s.S.SpanTrips(from - 60*86400)
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	cs, err := s.S.SpanCharges(from - 86400)
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	named, _ := s.S.ListNamedPlaces()
	ok(w, timeuse.Analyze(ts, cs, named, from, now.Unix(), s.S.FirstSampleTS(), s.defaultLoc()))
}
