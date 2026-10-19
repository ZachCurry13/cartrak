package server

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

func (s *Server) listTimeOff(w http.ResponseWriter, r *http.Request) {
	ts, err := s.S.ListTimeOff()
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, map[string]any{"items": ts, "today": time.Now().In(s.defaultLoc()).Format("2006-01-02")})
}

// addTimeOff saves a vacation. Every trip that starts on those days (in the car's time zone) is Personal, and trips
// already recorded in the stretch are relabelled at once. Admin only.
func (s *Server) addTimeOff(w http.ResponseWriter, r *http.Request) {
	var in store.TimeOff
	if decode(r, &in) != nil {
		fail(w, 400, "bad request")
		return
	}
	a, e1 := time.Parse("2006-01-02", in.From)
	b, e2 := time.Parse("2006-01-02", in.To)
	if e1 != nil || e2 != nil {
		fail(w, 400, "choose the first and last day")
		return
	}
	if b.Before(a) {
		fail(w, 400, "the last day must not be before the first")
		return
	}
	if b.Sub(a) > 366*24*time.Hour {
		fail(w, 400, "a vacation can be at most a year")
		return
	}
	in.Label = strings.TrimSpace(in.Label)
	if len(in.Label) > 60 {
		in.Label = in.Label[:60]
	}
	if err := s.S.AddTimeOff(&in); err != nil {
		fail(w, 500, "db error")
		return
	}
	s.reclassifyAfterTimeOff()
	ok(w, in)
}

func (s *Server) deleteTimeOff(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err := s.S.DeleteTimeOff(id); err != nil {
		fail(w, 500, "db error")
		return
	}
	s.reclassifyAfterTimeOff()
	ok(w, map[string]any{"deleted": id})
}

func (s *Server) reclassifyAfterTimeOff() {
	if s.D != nil {
		_ = s.D.Reclassify() // trips set by hand are left alone
	}
}
