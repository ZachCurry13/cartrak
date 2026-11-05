package server

import (
	"context"
	"net/http"
	"time"

	"github.com/ZachCurry13/cartrak/internal/geocode"
	"github.com/ZachCurry13/cartrak/internal/googleplaces"
)

// searchPlaces turns a typed address into a few places to choose from: GET /api/named-places/search?q=... Admin only. The address
// goes to the city lookup service (OpenStreetMap by default) and is not kept, and the lookup stops when City names is switched off.
func (s *Server) searchPlaces(w http.ResponseWriter, r *http.Request) {
	c := s.geoClient()
	gkey := ""
	if s.Cfg != nil {
		gkey = s.Cfg.GooglePlacesKey()
	}
	if c == nil && gkey == "" {
		fail(w, 400, "Address lookup is off: City names is switched off in Settings, so nothing is sent to OpenStreetMap")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	var near *[2]float64 // where the car is, so matches in your own region come first
	if x, _ := s.S.LatestSample(); x != nil && x.Lat != nil && x.Lon != nil {
		near = &[2]float64{*x.Lat, *x.Lon}
	}
	var found []geocode.Match
	var err error
	if gkey != "" { // Google knows far more businesses than the map's own search
		region := ""
		if s.Cfg.LiveGasUnits() {
			region = "US"
		}
		found, err = googleplaces.Search(ctx, nil, s.PlacesURL, gkey, r.URL.Query().Get("q"), region, near)
	}
	if (gkey == "" || err != nil || len(found) == 0) && c != nil {
		found, err = c.Search(ctx, r.URL.Query().Get("q"), near)
	}
	if err != nil {
		fail(w, upstreamFailed, "The address lookup did not answer: "+err.Error())
		return
	}
	ok(w, found)
}
