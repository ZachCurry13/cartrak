package server

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/ZachCurry13/cartrak/internal/charging"
	"github.com/ZachCurry13/cartrak/internal/journey"
	"github.com/ZachCurry13/cartrak/internal/store"
)

func (s *Server) listJourneys(w http.ResponseWriter, r *http.Request) {
	js, err := s.S.ListJourneys()
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, js)
}

// checkGroupable reports a problem with trips that are about to join journey `into` (0 for a new one).
func checkGroupable(ts []*store.Trip, want int, into int64) string {
	if len(ts) != want {
		return "one of those trips no longer exists"
	}
	for _, t := range ts {
		if t.Open || t.EndTS == nil {
			return "a trip that is still in progress cannot be added"
		}
		if t.JourneyID != nil && *t.JourneyID != into {
			return "a trip is already in another journey"
		}
	}
	return ""
}

func cleanName(n string) string {
	n = strings.TrimSpace(n)
	if len(n) > 80 {
		n = n[:80]
	}
	if n == "" {
		return "Journey"
	}
	return n
}

// createJourney groups two or more finished trips. Admin only.
func (s *Server) createJourney(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name    string  `json:"name"`
		TripIDs []int64 `json:"tripIds"`
	}
	if decode(r, &in) != nil || len(in.TripIDs) < 2 || len(in.TripIDs) > 200 {
		fail(w, 400, "choose two or more trips")
		return
	}
	ts, err := s.S.TripsByIDs(in.TripIDs)
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	if msg := checkGroupable(ts, len(in.TripIDs), 0); msg != "" {
		fail(w, http.StatusConflict, msg)
		return
	}
	id, err := s.S.CreateJourney(cleanName(in.Name))
	if err == nil {
		err = s.S.SetTripsJourney(in.TripIDs, id)
	}
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, map[string]any{"id": id})
}

// journeyDetail is a journey with its legs, totals, stops and cost.
func (s *Server) journeyDetail(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	j, err := s.S.GetJourney(id)
	if err != nil || j == nil {
		fail(w, 404, "no such journey")
		return
	}
	ts, err := s.S.JourneyTrips(id)
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	var sessions []charging.Session
	if all, err := s.chargeSessions(1000); err == nil {
		for _, c := range all {
			if c.StartTS >= j.StartTS-3600 && c.StartTS <= j.EndTS+7200 {
				sessions = append(sessions, c)
			}
		}
	}
	p := s.prices()
	ok(w, map[string]any{
		"journey": j, "trips": s.tripsWithEnergy(ts), "currency": s.currency(),
		"summary": journey.Summarize(ts, s.capacityKwh(), sessions, p.Public),
	})
}

// patchJourney renames a journey or adds and removes trips. Admin only.
func (s *Server) patchJourney(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	var in struct {
		Name   *string `json:"name"`
		Add    []int64 `json:"add"`
		Remove []int64 `json:"remove"`
	}
	if decode(r, &in) != nil {
		fail(w, 400, "bad request")
		return
	}
	if j, err := s.S.GetJourney(id); err != nil || j == nil {
		fail(w, 404, "no such journey")
		return
	}
	if len(in.Add) > 0 {
		ts, err := s.S.TripsByIDs(in.Add)
		if err != nil {
			fail(w, 500, "db error")
			return
		}
		if msg := checkGroupable(ts, len(in.Add), id); msg != "" {
			fail(w, http.StatusConflict, msg)
			return
		}
		_ = s.S.SetTripsJourney(in.Add, id)
	}
	if len(in.Remove) > 0 {
		own, _ := s.S.JourneyTrips(id)
		mine := map[int64]bool{}
		for _, t := range own {
			mine[t.ID] = true
		}
		var ids []int64
		for _, t := range in.Remove {
			if mine[t] {
				ids = append(ids, t)
			}
		}
		_ = s.S.SetTripsJourney(ids, 0)
	}
	if in.Name != nil {
		_ = s.S.RenameJourney(id, cleanName(*in.Name))
	}
	ok(w, "saved")
}

// deleteJourney ungroups the trips. They are not deleted. Admin only.
func (s *Server) deleteJourney(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err := s.S.DeleteJourney(id); err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, "ungrouped")
}
