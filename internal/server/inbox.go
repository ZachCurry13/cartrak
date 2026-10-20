package server

import "net/http"

// InboxItem is one alert in the inbox, with the car it was about.
type InboxItem struct {
	Car     int64  `json:"car"`
	CarName string `json:"carName"` // only when the person can open more than one car
	ID      int64  `json:"id"`
	TS      int64  `json:"ts"`
	Kind    string `json:"kind"`
	Title   string `json:"title"`
	Body    string `json:"body"`
	URL     string `json:"url"`
}

// inbox: GET /api/inbox. The latest alerts about every car the person can open, newest first, whether or not they have
// notifications switched on for a device. It is what the bell in the header shows.
func (s *Server) inbox(w http.ResponseWriter, r *http.Request) {
	const limit = 100
	if s.Cars != nil {
		ok(w, map[string]any{"items": s.Cars.Inbox(userFrom(r).ID, limit)})
		return
	}
	xs, err := s.S.RecentAlerts(limit)
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	out := make([]InboxItem, 0, len(xs))
	for _, x := range xs {
		out = append(out, InboxItem{ID: x.ID, TS: x.TS, Kind: x.Kind, Title: x.Title, Body: x.Body, URL: x.URL})
	}
	ok(w, map[string]any{"items": out})
}
