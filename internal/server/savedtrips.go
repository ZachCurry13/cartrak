package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	savedTripMax  = 60
	savedTripSize = 200 << 10
)

// listSavedTrips: GET /api/saved-trips. Admin only.
func (s *Server) listSavedTrips(w http.ResponseWriter, r *http.Request) {
	t, err := s.S.SavedTrips()
	if err != nil {
		fail(w, http.StatusInternalServerError, err.Error())
		return
	}
	shares, _ := s.S.TripShares()
	out := make([]savedTripOut, len(t))
	for i, x := range t {
		out[i] = savedTripOut{SavedTrip: x}
		if sh, on := shares[x.ID]; on && sh.Expires > time.Now().Unix() {
			out[i].Share = &tripShareOut{Expires: sh.Expires, Opens: sh.Opens, HideStart: sh.HideStart, URL: s.publicBase() + "/t/" + sh.Token}
		}
	}
	ok(w, out)
}

// addSavedTrip: POST /api/saved-trips {"name": "...", "body": {...}}. The body is whatever the page needs to open the trip again.
func (s *Server) addSavedTrip(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name string          `json:"name"`
		Body json.RawMessage `json:"body"`
	}
	if decode(r, &in) != nil || strings.TrimSpace(in.Name) == "" || len(in.Name) > 80 || len(in.Body) == 0 || len(in.Body) > savedTripSize || !json.Valid(in.Body) {
		fail(w, 400, "give the trip a name")
		return
	}
	if all, _ := s.S.SavedTrips(); len(all) >= savedTripMax {
		fail(w, 400, "That is a lot of saved trips: delete some first")
		return
	}
	t, err := s.S.AddSavedTrip(strings.TrimSpace(in.Name), string(in.Body), time.Now().Unix())
	if err != nil {
		fail(w, http.StatusInternalServerError, err.Error())
		return
	}
	ok(w, t)
}

// deleteSavedTrip: DELETE /api/saved-trips/{id}.
func (s *Server) deleteSavedTrip(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		fail(w, 400, "no such trip")
		return
	}
	if err := s.S.DeleteSavedTrip(id); err != nil {
		fail(w, http.StatusInternalServerError, err.Error())
		return
	}
	ok(w, map[string]bool{"ok": true})
}
