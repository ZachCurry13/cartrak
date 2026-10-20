package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
	"golang.org/x/crypto/bcrypt"
)

const (
	cookieName        = "cartrak_session"
	defaultSessionTTL = 30 * 24 * time.Hour
)

type ctxKey struct{}

type siteKey struct{}

// siteAdmin says whether the signed-in person administers the whole installation, whatever their role for the car on
// screen. A car's owner is an administrator of that car only.
func siteAdmin(r *http.Request) bool {
	v, _ := r.Context().Value(siteKey{}).(bool)
	return v
}

// shortToken is a short secret for a public link: n characters of lowercase letters and digits (5 bits each, so 12 is 60 bits). That is far
// more than can be guessed through the limit on wrong links, and short enough to read out or type.
func shortToken(n int) string {
	const alphabet = "abcdefghijklmnopqrstuvwxyz234567"
	b := make([]byte, n)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = alphabet[int(b[i])%len(alphabet)] // 256 is a multiple of 32, so every letter is equally likely
	}
	return string(b)
}

func randToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// secure reports whether the browser reached us over HTTPS (directly or via
// the Cloudflare tunnel), so the cookie can be marked Secure.
func secure(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

func (s *Server) startSession(w http.ResponseWriter, r *http.Request, u *store.User) error {
	tok := randToken()
	if err := s.core().NewSession(tok, u.ID, s.sessionTTL()); err != nil {
		return err
	}
	_ = s.core().SetSessionInfo(tok, s.clientIP(r), describeAgent(r.UserAgent()))
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: tok, Path: "/", HttpOnly: true,
		SameSite: http.SameSiteLaxMode, Secure: secure(r), MaxAge: int(s.sessionTTL().Seconds())})
	return nil
}

func userFrom(r *http.Request) *store.User {
	u, _ := r.Context().Value(ctxKey{}).(*store.User)
	return u
}

// auth requires a valid session and access to this server's car; admin additionally requires being an administrator of
// that car: a site administrator, or the person who owns the car.
func (s *Server) auth(admin bool, next http.HandlerFunc) http.HandlerFunc {
	return s.guard(admin, false, next)
}

// authSite is for what belongs to the whole installation (people, sign-in log, notification devices, remote access): a
// valid session is enough unless admin is set, which then needs the site administrator role.
func (s *Server) authSite(admin bool, next http.HandlerFunc) http.HandlerFunc {
	return s.guard(admin, true, next)
}

// carRole is the role a person has for this server's car: "admin" for a site administrator or the car's owner, "viewer"
// for someone it was shared with, "" for no access. With a single car (CarID 0) the person's own role stands.
func (s *Server) carRole(u *store.User) string {
	if s.CarID == 0 || u.Role == "admin" {
		return u.Role
	}
	switch s.core().Access(u.ID, s.CarID) {
	case store.AccessOwner:
		return "admin"
	case store.AccessViewer:
		return "viewer"
	}
	return ""
}

func (s *Server) guard(admin, site bool, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(cookieName)
		if err != nil {
			fail(w, 401, "login required")
			return
		}
		u, err := s.core().SessionUser(c.Value)
		if err != nil || u == nil {
			fail(w, 401, "login required")
			return
		}
		isSite := u.Role == "admin"
		if !site {
			role := s.carRole(u)
			if role == "" {
				fail(w, 403, "you do not have access to this car")
				return
			}
			if u.Role == "admin" && s.CarID != 0 && s.core().Access(u.ID, s.CarID) == "" {
				s.noteAdminView(u.Name) // an administrator looking at a car that is not theirs: its owner is told
			}
			eu := *u
			eu.Role = role
			u = &eu
		}
		if admin && u.Role != "admin" {
			fail(w, 403, "admin only")
			return
		}
		// State-changing calls must come from our own pages (CSRF defence).
		if r.Method != http.MethodGet && r.Header.Get("X-Requested-With") != "cartrak" {
			fail(w, 403, "missing header")
			return
		}
		ctx := context.WithValue(r.Context(), ctxKey{}, u)
		next(w, r.WithContext(context.WithValue(ctx, siteKey{}, isSite)))
	}
}

func hashPassword(p string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(p), bcrypt.DefaultCost)
	return string(b), err
}

// limiter blocks an IP for a minute after 5 failed logins.
type limiter struct {
	mu    sync.Mutex
	fails map[string][]time.Time
}

func (l *limiter) blocked(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	cut := time.Now().Add(-time.Minute)
	var keep []time.Time
	for _, t := range l.fails[ip] {
		if t.After(cut) {
			keep = append(keep, t)
		}
	}
	if l.fails == nil {
		l.fails = map[string][]time.Time{}
	}
	l.fails[ip] = keep
	return len(keep) >= 5
}

func (l *limiter) fail(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.fails[ip] = append(l.fails[ip], time.Now())
}

func (s *Server) sessionTTL() time.Duration {
	if s.SessionDays > 0 {
		return time.Duration(s.SessionDays) * 24 * time.Hour
	}
	return defaultSessionTTL
}

// clientIP returns the caller's address. Only when TrustProxy is set do we
// believe the proxy headers (Cloudflare sends CF-Connecting-IP); otherwise
// anyone could spoof them to dodge the login lockout.
func (s *Server) clientIP(r *http.Request) string {
	if s.TrustProxy {
		if ip := strings.TrimSpace(r.Header.Get("CF-Connecting-IP")); ip != "" {
			return ip
		}
		if ip := strings.TrimSpace(r.Header.Get("X-Real-IP")); ip != "" {
			return ip
		}
		if xf := r.Header.Get("X-Forwarded-For"); xf != "" {
			return strings.TrimSpace(strings.Split(xf, ",")[0])
		}
	}
	h, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return h
}

// randHex returns 2*n random hex characters. Ford's sign-in rejects a state longer than 16
// characters, so the link uses randHex(8).
func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
