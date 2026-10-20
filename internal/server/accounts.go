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
	OwnCar   bool   `json:"ownCar"`
	SeeMain  *bool  `json:"seeMain"` // read access to the first car; on unless said otherwise
}

// setupState tells the UI whether first-run setup is still needed.
func (s *Server) setupState(w http.ResponseWriter, r *http.Request) {
	ok(w, map[string]bool{"needsSetup": s.core().CountUsers() == 0})
}

// setup creates the first admin. It only works while there are no users.
func (s *Server) setup(w http.ResponseWriter, r *http.Request) {
	if s.core().CountUsers() != 0 {
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
	u, _ := s.core().UserByName(c.Name)
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
	if err := s.core().CreateUser(strings.TrimSpace(name), h, role); err != nil {
		return err
	}
	if role == "admin" { // the first person to sign up, or one made by the seed, manages the first car
		s.grantNewUser(name, role, true)
	}
	return nil
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
	u, _ := s.core().UserByName(strings.TrimSpace(c.Name))
	if u == nil || checkPassword(u.Hash, c.Password) != nil {
		s.lim.fail(ip)
		s.logSignin(r, strings.TrimSpace(c.Name), false)
		fail(w, 401, "wrong name or password")
		return
	}
	if err := s.startSession(w, r, u); err != nil {
		fail(w, 500, "session error")
		return
	}
	s.logSignin(r, u.Name, true)
	ok(w, u)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(cookieName); err == nil {
		s.core().DeleteSession(c.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: cookieName, Path: "/", MaxAge: -1})
	ok(w, "bye")
}

func (s *Server) listUsers(w http.ResponseWriter, r *http.Request) {
	us, err := s.core().ListUsers()
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
	if u, _ := s.core().UserByName(strings.TrimSpace(c.Name)); u != nil && c.OwnCar {
		_ = s.core().SetOwnCar(u.ID, true)
	}
	s.grantNewUser(strings.TrimSpace(c.Name), c.Role, c.SeeMain == nil || *c.SeeMain)
	ok(w, "created")
}

func (s *Server) deleteUser(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if id == userFrom(r).ID {
		fail(w, 400, "you cannot delete yourself")
		return
	}
	if err := s.core().DeleteUser(id); err != nil {
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
			if ok, _ := s.core().SetPassword(name, h); !ok && s.core().CountUsers() == 0 {
				_ = s.makeUser(name, pass, "admin")
			}
		}
		return
	}
	if s.core().CountUsers() == 0 {
		_ = s.makeUser(name, pass, "admin")
	}
}
