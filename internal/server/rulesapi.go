package server

import (
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
	if ru.Name == "" || !s.S.CategoryExists(ru.Category) ||
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
	if s.Cfg != nil {
		if !s.Cfg.Configured() { // credentials missing: send the admin to enter them
			http.Redirect(w, r, "/#/admin", http.StatusFound)
			return
		}
		if v, src := s.Cfg.Get("FORD_REDIRECT_URL"); v == "" || src == "default" {
			// No redirect URL chosen: use the address this browser is on. It is
			// stored so the code exchange sends exactly the same value.
			_ = s.S.SetKV("derived_redirect", s.origin(r)+"/ford/callback")
		}
	}
	if !s.ensureRedirect(r) { // Ford accepts none of our candidate addresses: show why here, not on a Ford error page
		http.Redirect(w, r, "/#/connect", http.StatusFound)
		return
	}
	state := s.fordState() // Ford requires a state of at most 16 characters
	_ = s.S.SetKV("ford_state", state+"|"+strconv.FormatInt(time.Now().Add(15*time.Minute).Unix(), 10))
	http.Redirect(w, r, s.Link.LinkURL(state), http.StatusFound)
}
