package server

import (
	"net/http"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

// A person's own birthday and holiday touches. GET and PUT /api/me/prefs; only the person signed in sees or changes theirs.

func validBirthday(b string) bool {
	if b == "" {
		return true
	}
	t, err := time.Parse("01-02", b)
	return err == nil && t.Format("01-02") == b
}

func (s *Server) getPrefs(w http.ResponseWriter, r *http.Request) {
	ok(w, s.core().Prefs(userFrom(r).ID))
}

func (s *Server) putPrefs(w http.ResponseWriter, r *http.Request) {
	var p store.UserPrefs
	if decode(r, &p) != nil || !validBirthday(p.Birthday) {
		fail(w, 400, "the birthday is a month and a day, like 07-14")
		return
	}
	if err := s.core().SetPrefs(userFrom(r).ID, p); err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, p)
}
