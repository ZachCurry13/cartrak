package server

import "net/http"

// parkedWatchInfo is the Parked watch page: the bump level, how many parked readings were checked and the strongest
// motion seen, and the recent events. GET /api/parked-watch.
func (s *Server) parkedWatchInfo(w http.ResponseWriter, r *http.Request) {
	events, err := s.S.RecentParkedEvents(50)
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	level := 1.0
	if s.Cfg != nil {
		level = s.Cfg.BumpThreshold()
	}
	ok(w, map[string]any{"level": level, "on": level > 0, "stats": s.S.ParkedWatchStats(), "events": events})
}

// parkedWatchReset clears the totals and the list, to start a fresh trial. Admin only.
func (s *Server) parkedWatchReset(w http.ResponseWriter, r *http.Request) {
	if err := s.S.ResetParkedWatch(); err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, "reset")
}
