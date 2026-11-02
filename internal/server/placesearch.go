package server

import (
	"context"
	"net/http"
	"time"
)

// searchPlaces turns a typed address into a few places to choose from: GET /api/named-places/search?q=... Admin only. The address
// goes to the city lookup service (OpenStreetMap by default) and is not kept, and the lookup stops when City names is switched off.
func (s *Server) searchPlaces(w http.ResponseWriter, r *http.Request) {
	c := s.geoClient()
	if c == nil {
		fail(w, 400, "Address lookup is off: City names is switched off in Settings, so nothing is sent to OpenStreetMap")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	found, err := c.Search(ctx, r.URL.Query().Get("q"))
	if err != nil {
		fail(w, 502, "The address lookup did not answer: "+err.Error())
		return
	}
	ok(w, found)
}
