package server

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/ZachCurry13/cartrak/internal/places"
	"github.com/ZachCurry13/cartrak/internal/store"
)

// placeNames fills in the names of the places a trip started and ended at.
func (s *Server) placeNames(o tripOut) tripOut {
	all, _ := s.S.ListNamedPlaces()
	if len(all) == 0 || o.Trip == nil {
		return o
	}
	o.StartPlace = places.Name(o.StartLat, o.StartLon, all)
	o.EndPlace = places.Name(o.EndLat, o.EndLon, all)
	return o
}

func (s *Server) listNamedPlaces(w http.ResponseWriter, r *http.Request) {
	ps, err := s.S.ListNamedPlaces()
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, ps)
}

type placeIn struct {
	Name    *string  `json:"name"`
	RadiusM *float64 `json:"radiusM"`
	Lat     *float64 `json:"lat"`
	Lon     *float64 `json:"lon"`
	TripID  int64    `json:"tripId"`
	End     string   `json:"end"`  // "start" or "end" of that trip
	Here    bool     `json:"here"` // where the car is now
}

func (in *placeIn) check() string {
	if in.Name != nil && (strings.TrimSpace(*in.Name) == "" || len(*in.Name) > 40) {
		return "give the place a short name"
	}
	if in.RadiusM != nil && (*in.RadiusM < 50 || *in.RadiusM > 2000) {
		return "the size must be between 50 m and 2000 m"
	}
	return ""
}

// addNamedPlace names a place from a trip's start or end, from where the car is now, or from coordinates. Admin only.
func (s *Server) addNamedPlace(w http.ResponseWriter, r *http.Request) {
	var in placeIn
	if decode(r, &in) != nil || in.Name == nil {
		fail(w, 400, "give the place a name")
		return
	}
	if msg := in.check(); msg != "" {
		fail(w, 400, msg)
		return
	}
	lat, lon := in.Lat, in.Lon
	switch {
	case in.TripID > 0:
		t, err := s.S.GetTrip(in.TripID)
		if err != nil || t == nil {
			fail(w, 404, "no such trip")
			return
		}
		lat, lon = t.EndLat, t.EndLon
		if in.End == "start" {
			lat, lon = t.StartLat, t.StartLon
		}
	case in.Here:
		if x, _ := s.S.LatestSample(); x != nil {
			lat, lon = x.Lat, x.Lon
		}
	}
	if lat == nil || lon == nil || *lat < -90 || *lat > 90 || *lon < -180 || *lon > 180 {
		fail(w, 400, "CarTrak does not know where that is")
		return
	}
	p := &store.NamedPlace{Name: strings.TrimSpace(*in.Name), Lat: *lat, Lon: *lon, RadiusM: 150}
	if in.RadiusM != nil {
		p.RadiusM = *in.RadiusM
	}
	if err := s.S.SaveNamedPlace(p); err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, map[string]any{"id": p.ID})
}

func (s *Server) patchNamedPlace(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	var in placeIn
	p, err := s.S.GetNamedPlace(id)
	if decode(r, &in) != nil || err != nil || p == nil {
		fail(w, 404, "no such place")
		return
	}
	if msg := in.check(); msg != "" {
		fail(w, 400, msg)
		return
	}
	if in.Name != nil {
		p.Name = strings.TrimSpace(*in.Name)
	}
	if in.RadiusM != nil {
		p.RadiusM = *in.RadiusM
	}
	if err := s.S.SaveNamedPlace(p); err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, "saved")
}

func (s *Server) deleteNamedPlace(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err := s.S.DeleteNamedPlace(id); err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, "deleted")
}
