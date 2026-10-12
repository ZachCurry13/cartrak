package server

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/ZachCurry13/cartrak/internal/store"
)

type creds struct {
	Name     string `json:"name"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

// setupState tells the UI whether first-run setup is still needed.
func (s *Server) setupState(w http.ResponseWriter, r *http.Request) {
	ok(w, map[string]bool{"needsSetup": s.S.CountUsers() == 0})
}

// setup creates the first admin. It only works while there are no users.
func (s *Server) setup(w http.ResponseWriter, r *http.Request) {
	if s.S.CountUsers() != 0 {
		fail(w, 403, "already set up")
		return
	}
	var c creds
	if decode(r, &c) != nil || !validCreds(c) {
		fail(w, 400, "name required and password must be 10+ characters")
		return
	}
	if err := s.makeUser(c.Name, c.Password, "admin"); err != nil {
		fail(w, 500, "could not create user")
		return
	}
	u, _ := s.S.UserByName(c.Name)
	if err := s.startSession(w, r, u); err != nil {
		fail(w, 500, "session error")
		return
	}
	ok(w, u)
}

func validCreds(c creds) bool {
	return strings.TrimSpace(c.Name) != "" && len(c.Password) >= 10 && len(c.Password) <= 200
}

func (s *Server) makeUser(name, pass, role string) error {
	h, err := hashPassword(pass)
	if err != nil {
		return err
	}
	return s.S.CreateUser(strings.TrimSpace(name), h, role)
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	ip := s.clientIP(r)
	if s.lim.blocked(ip) {
		fail(w, 429, "too many attempts, wait a minute")
		return
	}
	var c creds
	if decode(r, &c) != nil {
		fail(w, 400, "bad request")
		return
	}
	u, _ := s.S.UserByName(strings.TrimSpace(c.Name))
	if u == nil || checkPassword(u.Hash, c.Password) != nil {
		s.lim.fail(ip)
		fail(w, 401, "wrong name or password")
		return
	}
	if err := s.startSession(w, r, u); err != nil {
		fail(w, 500, "session error")
		return
	}
	ok(w, u)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(cookieName); err == nil {
		s.S.DeleteSession(c.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: cookieName, Path: "/", MaxAge: -1})
	ok(w, "bye")
}

func (s *Server) listUsers(w http.ResponseWriter, r *http.Request) {
	us, err := s.S.ListUsers()
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, us)
}

func (s *Server) addUser(w http.ResponseWriter, r *http.Request) {
	var c creds
	if decode(r, &c) != nil || !validCreds(c) || (c.Role != "admin" && c.Role != "viewer") {
		fail(w, 400, "name, 10+ char password and role (admin|viewer) required")
		return
	}
	if err := s.makeUser(c.Name, c.Password, c.Role); err != nil {
		fail(w, 409, "that name is taken")
		return
	}
	ok(w, "created")
}

func (s *Server) deleteUser(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if id == userFrom(r).ID {
		fail(w, 400, "you cannot delete yourself")
		return
	}
	if err := s.S.DeleteUser(id); err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, "deleted")
}

var _ = store.User{}

// SeedAdmin creates the first admin from settings if there are no users. With
// reset set, it instead changes that user's password (account recovery).
func (s *Server) SeedAdmin(name, pass string, reset bool) {
	if pass == "" {
		return
	}
	if reset {
		if h, err := hashPassword(pass); err == nil {
			if ok, _ := s.S.SetPassword(name, h); !ok && s.S.CountUsers() == 0 {
				_ = s.makeUser(name, pass, "admin")
			}
		}
		return
	}
	if s.S.CountUsers() == 0 {
		_ = s.makeUser(name, pass, "admin")
	}
}
