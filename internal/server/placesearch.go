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
	var near *[2]float64 // where the car is, so matches in your own region come first
	if x, _ := s.S.LatestSample(); x != nil && x.Lat != nil && x.Lon != nil {
		near = &[2]float64{*x.Lat, *x.Lon}
	}
	found, err := c.Search(ctx, r.URL.Query().Get("q"), near)
	if err != nil {
		fail(w, upstreamFailed, "The address lookup did not answer: "+err.Error())
		return
	}
	ok(w, found)
}
