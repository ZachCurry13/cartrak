// Package server exposes the JSON API and the embedded mobile web app.
package server

import (
	"context"
	"encoding/json"
	"io/fs"
	"net/http"

	"github.com/ZachCurry13/cartrak/internal/poller"
	"github.com/ZachCurry13/cartrak/internal/store"
	"github.com/ZachCurry13/cartrak/internal/trips"
	"github.com/ZachCurry13/cartrak/internal/updater"
	"github.com/ZachCurry13/cartrak/internal/web"
)

// Linker is implemented by providers that need the owner to sign in at Ford.
type Linker interface {
	LinkURL(state string) string
	Exchange(ctx context.Context, code string) error
}

type Server struct {
	S       *store.Store
	P       *poller.Poller
	D       *trips.Detector
	Link    Linker // nil when the provider needs no account link
	Version string
	Upd     *updater.Checker // nil disables update checks
	lim     limiter
}

func (s *Server) Handler() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })

	m.HandleFunc("GET /api/setup", s.setupState)
	m.HandleFunc("POST /api/setup", s.setup)
	m.HandleFunc("POST /api/login", s.login)
	m.HandleFunc("POST /api/logout", s.auth(false, s.logout))
	m.HandleFunc("GET /api/me", s.auth(false, func(w http.ResponseWriter, r *http.Request) { ok(w, userFrom(r)) }))

	m.HandleFunc("GET /api/about", s.auth(false, s.about))
	m.HandleFunc("POST /api/settings/update-check", s.auth(true, s.setUpdateCheck))
	m.HandleFunc("GET /api/changelog", s.auth(false, s.changelog))
	m.HandleFunc("GET /api/onboarding", s.auth(false, s.onboarding))
	m.HandleFunc("POST /api/onboarding/dismiss", s.auth(true, s.dismissOnboarding))

	m.HandleFunc("GET /api/status", s.auth(false, s.status))
	m.HandleFunc("GET /api/latest", s.auth(false, s.latest))
	m.HandleFunc("GET /api/trips", s.auth(false, s.listTrips))
	m.HandleFunc("GET /api/trips/{id}", s.auth(false, s.tripDetail))
	m.HandleFunc("PATCH /api/trips/{id}", s.auth(true, s.patchTrip))
	m.HandleFunc("GET /api/charges", s.auth(false, s.listCharges))
	m.HandleFunc("GET /api/summary", s.auth(false, s.summary))
	m.HandleFunc("GET /api/export/trips.csv", s.auth(false, s.exportCSV))

	m.HandleFunc("GET /api/rules", s.auth(false, s.listRules))
	m.HandleFunc("POST /api/rules", s.auth(true, s.saveRule))
	m.HandleFunc("PUT /api/rules/{id}", s.auth(true, s.saveRule))
	m.HandleFunc("DELETE /api/rules/{id}", s.auth(true, s.deleteRule))
	m.HandleFunc("POST /api/rules/apply", s.auth(true, s.applyRules))

	m.HandleFunc("GET /api/users", s.auth(true, s.listUsers))
	m.HandleFunc("POST /api/users", s.auth(true, s.addUser))
	m.HandleFunc("DELETE /api/users/{id}", s.auth(true, s.deleteUser))

	m.HandleFunc("GET /ford/connect", s.auth(true, s.fordConnect))
	m.HandleFunc("GET /ford/callback", s.auth(true, s.fordCallback))
	m.HandleFunc("POST /api/poll", s.auth(true, func(w http.ResponseWriter, r *http.Request) { s.P.Now(); ok(w, "queued") }))

	static, _ := fs.Sub(web.Files, "static")
	m.Handle("/", noCacheHTML(http.FileServer(http.FS(static))))
	return secHeaders(m)
}

func secHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "same-origin")
		h.Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}

// noCacheHTML keeps pages and the service worker fresh; assets are versioned by sw.js.
func noCacheHTML(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		next.ServeHTTP(w, r)
	})
}

func ok(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func fail(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func decode(r *http.Request, v any) error {
	return json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20)).Decode(v)
}
