package server

import (
	"crypto/subtle"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
	"golang.org/x/crypto/bcrypt"
)

func checkPassword(hash, pass string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pass))
}

func (s *Server) listRules(w http.ResponseWriter, r *http.Request) {
	rs, err := s.S.ListRules()
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, rs)
}

func (s *Server) saveRule(w http.ResponseWriter, r *http.Request) {
	var ru store.Rule
	if decode(r, &ru) != nil {
		fail(w, 400, "bad request")
		return
	}
	if id := r.PathValue("id"); id != "" {
		ru.ID, _ = strconv.ParseInt(id, 10, 64)
	}
	ru.Name = strings.TrimSpace(ru.Name)
	if ru.Name == "" || (ru.Category != "work" && ru.Category != "personal") ||
		ru.Days < 0 || ru.Days > 127 || ru.StartMin < 0 || ru.StartMin > 1440 || ru.EndMin < 0 || ru.EndMin > 1440 {
		fail(w, 400, "invalid rule")
		return
	}
	id, err := s.S.SaveRule(ru)
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	ru.ID = id
	ok(w, ru)
}

func (s *Server) deleteRule(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err := s.S.DeleteRule(id); err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, "deleted")
}

// applyRules re-runs the rules over past trips (manual choices are kept).
func (s *Server) applyRules(w http.ResponseWriter, r *http.Request) {
	if err := s.D.Reclassify(); err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, "applied")
}

// fordConnect sends the admin to Ford to sign in and authorize the vehicle.
func (s *Server) fordConnect(w http.ResponseWriter, r *http.Request) {
	if s.Link == nil {
		fail(w, 404, "this provider needs no link")
		return
	}
	state := randToken()
	_ = s.S.SetKV("ford_state", state+"|"+strconv.FormatInt(time.Now().Add(15*time.Minute).Unix(), 10))
	http.Redirect(w, r, s.Link.LinkURL(state), http.StatusFound)
}

func (s *Server) fordCallback(w http.ResponseWriter, r *http.Request) {
	if s.Link == nil {
		fail(w, 404, "no link configured")
		return
	}
	parts := strings.SplitN(s.S.GetKV("ford_state"), "|", 2)
	exp := int64(0)
	if len(parts) == 2 {
		exp, _ = strconv.ParseInt(parts[1], 10, 64)
	}
	got := r.URL.Query().Get("state")
	if len(parts) != 2 || time.Now().Unix() > exp || subtle.ConstantTimeCompare([]byte(parts[0]), []byte(got)) != 1 {
		fail(w, 400, "link expired or state mismatch, start again")
		return
	}
	_ = s.S.SetKV("ford_state", "")
	code := r.URL.Query().Get("code")
	if code == "" {
		fail(w, 400, "Ford did not return a code")
		return
	}
	if err := s.Link.Exchange(r.Context(), code); err != nil {
		fail(w, 502, "could not finish linking with Ford")
		return
	}
	s.P.Now()
	http.Redirect(w, r, "/#/settings", http.StatusFound)
}
