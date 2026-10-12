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
	if err := s.S.NewSession(tok, u.ID, s.sessionTTL()); err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: tok, Path: "/", HttpOnly: true,
		SameSite: http.SameSiteLaxMode, Secure: secure(r), MaxAge: int(s.sessionTTL().Seconds())})
	return nil
}

func userFrom(r *http.Request) *store.User {
	u, _ := r.Context().Value(ctxKey{}).(*store.User)
	return u
}

// auth requires a valid session; admin additionally requires the admin role.
func (s *Server) auth(admin bool, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(cookieName)
		if err != nil {
			fail(w, 401, "login required")
			return
		}
		u, err := s.S.SessionUser(c.Value)
		if err != nil || u == nil {
			fail(w, 401, "login required")
			return
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
		next(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, u)))
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
