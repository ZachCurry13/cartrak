package server

import (
	"net/http"
	"strings"

	"github.com/ZachCurry13/cartrak/internal/charging"
)

// namedHome is the place the owner named Home, if any.
func (s *Server) namedHome() *charging.Point {
	ps, err := s.S.ListNamedPlaces()
	if err != nil {
		return nil
	}
	for _, p := range ps {
		if strings.EqualFold(strings.TrimSpace(p.Name), "home") {
			return &charging.Point{Lat: p.Lat, Lon: p.Lon}
		}
	}
	return nil
}

// chargingHome: GET /api/charging/home. Where CarTrak thinks home is and how it knows, so the Charging settings page can say.
func (s *Server) chargingHome(w http.ResponseWriter, r *http.Request) {
	source := "none"
	if s.namedHome() != nil {
		source = "place"
	} else if s.homePoint() != nil {
		source = "marked"
	}
	ok(w, map[string]any{"source": source})
}
