package server

import "net/http"

// alertLog is the recent alerts CarTrak decided to send and who they reached, to find out why a notification
// did or did not arrive. Admin only.
func (s *Server) alertLog(w http.ResponseWriter, r *http.Request) {
	xs, err := s.S.RecentAlerts(50)
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, xs)
}
