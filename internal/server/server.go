// Package server exposes the JSON API and the embedded mobile web app.
package server

import (
	"context"
	"encoding/json"
	"io/fs"
	"net/http"
	"time"

	"github.com/ZachCurry13/cartrak/internal/connector"
	"github.com/ZachCurry13/cartrak/internal/poller"
	"github.com/ZachCurry13/cartrak/internal/push"
	"github.com/ZachCurry13/cartrak/internal/settings"
	"github.com/ZachCurry13/cartrak/internal/store"
	"github.com/ZachCurry13/cartrak/internal/trips"
	"github.com/ZachCurry13/cartrak/internal/tunnel"
	"github.com/ZachCurry13/cartrak/internal/updater"
	"github.com/ZachCurry13/cartrak/internal/web"
)

// Linker is implemented by providers that need the owner to sign in at Ford.
type Linker interface {
	LinkURL(state string) string
	Exchange(ctx context.Context, code string) error
}

type Server struct {
	S           *store.Store
	P           *poller.Poller
	D           *trips.Detector
	Link        Linker // nil when the provider needs no account link
	Version     string
	Cfg         *settings.Settings // admin-panel settings (nil in some tests)
	Prober      Prober             // optional: Ford diagnostics
	Redirects   RedirectChecker    // optional: ask Ford which redirect addresses are registered
	Tunnel      *tunnel.Supervisor // in-app Cloudflare connector (nil in some tests)
	Conn        *connector.Manager // keeps the connector up to date (nil in some tests)
	Port        string             // port the app listens on, for the setup steps
	hub         hub
	TrustProxy  bool             // believe CF-Connecting-IP / X-Real-IP / X-Forwarded-For
	SessionDays int              // "keep me signed in" length; 0 = 30
	Upd         *updater.Checker // nil disables update checks
	lim         limiter
	devRate     rateWindow    // per-device call limit for the phone ingest API
	TileDir     string        // where downloaded map tiles are kept; empty switches maps off
	DataDir     string        // the data folder: backups are written here while they are made
	TileClient  *http.Client  // optional, for tests
	Push        *push.Service // Web Push alerts (nil in some tests)
	GeoGap      time.Duration // spacing of city lookups; 0 uses the service default (tests shorten it)
	geo         geoState
	tiles       tileFlight
}

func (s *Server) Handler() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })

	m.HandleFunc("GET /api/setup", s.setupState)
	m.HandleFunc("POST /api/setup", s.setup)
	m.HandleFunc("POST /api/login", s.login)
	m.HandleFunc("POST /api/logout", s.auth(false, s.logout))
	m.HandleFunc("GET /api/me", s.auth(false, func(w http.ResponseWriter, r *http.Request) { ok(w, userFrom(r)) }))

	m.HandleFunc("GET /api/admin/settings", s.auth(true, s.getSettings))
	m.HandleFunc("PUT /api/admin/settings", s.auth(true, s.saveSettings))
	m.HandleFunc("POST /api/admin/ford-probe", s.auth(true, s.fordProbe))
	m.HandleFunc("POST /api/admin/ford/detect-redirect", s.auth(true, s.detectRedirect))
	m.HandleFunc("GET /api/admin/remote", s.auth(true, s.getRemote))
	m.HandleFunc("PUT /api/admin/remote", s.auth(true, s.saveRemote))
	m.HandleFunc("GET /api/about", s.auth(false, s.about))
	m.HandleFunc("POST /api/settings/update-check", s.auth(true, s.setUpdateCheck))
	m.HandleFunc("GET /api/changelog", s.auth(false, s.changelog))
	m.HandleFunc("GET /api/onboarding", s.auth(false, s.onboarding))
	m.HandleFunc("POST /api/onboarding/dismiss", s.auth(true, s.dismissOnboarding))

	m.HandleFunc("GET /api/health", s.auth(false, s.health))
	m.HandleFunc("GET /api/vehicle", s.auth(false, s.vehicle))
	m.HandleFunc("GET /api/push/config", s.auth(false, s.pushConfig))
	m.HandleFunc("POST /api/push/subscribe", s.auth(false, s.pushSubscribe))
	m.HandleFunc("POST /api/push/prefs", s.auth(false, s.pushPrefs))
	m.HandleFunc("POST /api/push/unsubscribe", s.auth(false, s.pushUnsubscribe))
	m.HandleFunc("POST /api/push/test", s.auth(false, s.pushTest))
	m.HandleFunc("GET /api/mileage", s.auth(true, s.mileageReport))
	m.HandleFunc("GET /api/connector", s.auth(true, s.connectorInfo))
	m.HandleFunc("POST /api/connector/check", s.auth(true, s.connectorCheck))
	m.HandleFunc("POST /api/connector/update", s.auth(true, s.connectorUpdate))
	m.HandleFunc("GET /api/security", s.auth(true, s.securityInfo))
	m.HandleFunc("DELETE /api/security/sessions/{id}", s.auth(true, s.revokeSession))
	m.HandleFunc("GET /api/backup", s.auth(true, s.backupDownload))
	m.HandleFunc("GET /api/backup/info", s.auth(true, s.backupInfo))
	m.HandleFunc("GET /api/mileage/check", s.auth(true, s.mileageCheck))
	m.HandleFunc("POST /api/trips/missed", s.auth(true, s.addMissedTrip))
	m.HandleFunc("POST /api/trips/{id}/extend", s.auth(true, s.extendTrip))
	m.HandleFunc("GET /api/map", s.auth(false, s.mapConfig))
	m.HandleFunc("GET /api/tiles/{z}/{x}/{y}", s.auth(false, s.tile))
	m.HandleFunc("GET /api/vehicle-health", s.auth(false, s.vehicleHealth))
	m.HandleFunc("POST /api/health/battery12v-tested", s.auth(true, s.testedBattery))
	m.HandleFunc("DELETE /api/health/battery12v-tested", s.auth(true, s.testedBattery))
	m.HandleFunc("GET /api/car-data", s.auth(false, s.carData))
	m.HandleFunc("GET /api/range-band", s.auth(false, s.rangeBand))
	m.HandleFunc("GET /api/car-data/raw", s.auth(false, s.carDataRaw))
	m.HandleFunc("GET /api/timeline", s.auth(false, s.timeline))
	m.HandleFunc("GET /api/status", s.auth(false, s.status))
	m.HandleFunc("GET /api/latest", s.auth(false, s.latest))
	m.HandleFunc("GET /api/live", s.auth(false, s.liveStream))
	m.HandleFunc("POST /api/live-mode", s.auth(true, s.setLiveMode))
	m.HandleFunc("GET /api/trips", s.auth(false, s.listTrips))
	m.HandleFunc("GET /api/trips/routes", s.auth(false, s.tripRoutes))
	m.HandleFunc("GET /api/trips/{id}", s.auth(false, s.tripDetail))
	m.HandleFunc("GET /api/insights", s.auth(false, s.efficiencyInsights))
	m.HandleFunc("GET /api/time-use", s.auth(false, s.timeUse))
	m.HandleFunc("GET /api/named-places", s.auth(false, s.listNamedPlaces))
	m.HandleFunc("POST /api/named-places", s.auth(true, s.addNamedPlace))
	m.HandleFunc("PATCH /api/named-places/{id}", s.auth(true, s.patchNamedPlace))
	m.HandleFunc("DELETE /api/named-places/{id}", s.auth(true, s.deleteNamedPlace))
	m.HandleFunc("GET /api/maintenance", s.auth(false, s.listMaintenance))
	m.HandleFunc("POST /api/maintenance", s.auth(true, s.addMaintenance))
	m.HandleFunc("PATCH /api/maintenance/{id}", s.auth(true, s.patchMaintenance))
	m.HandleFunc("POST /api/maintenance/{id}/done", s.auth(true, s.maintenanceDone))
	m.HandleFunc("DELETE /api/maintenance/{id}", s.auth(true, s.deleteMaintenance))
	m.HandleFunc("GET /api/journeys", s.auth(false, s.listJourneys))
	m.HandleFunc("GET /api/shares", s.auth(true, s.listShares))
	m.HandleFunc("POST /api/shares", s.auth(true, s.createShare))
	m.HandleFunc("DELETE /api/shares/{id}", s.auth(true, s.revokeShare))
	m.HandleFunc("GET /s/{token}", s.sharePage)
	m.HandleFunc("GET /s/{token}/data", s.shareJSON)
	m.HandleFunc("GET /s/{token}/tile/{z}/{x}/{y}", s.shareTile)
	m.HandleFunc("POST /api/journeys", s.auth(true, s.createJourney))
	m.HandleFunc("GET /api/journeys/{id}", s.auth(false, s.journeyDetail))
	m.HandleFunc("PATCH /api/journeys/{id}", s.auth(true, s.patchJourney))
	m.HandleFunc("DELETE /api/journeys/{id}", s.auth(true, s.deleteJourney))
	m.HandleFunc("PATCH /api/trips/{id}", s.auth(true, s.patchTrip))
	m.HandleFunc("GET /api/charges", s.auth(false, s.listCharges))
	m.HandleFunc("GET /api/charges/summary", s.auth(false, s.chargeSummary))
	m.HandleFunc("PATCH /api/charges/{id}", s.auth(true, s.patchCharge))
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

	m.HandleFunc("GET /api/devices", s.auth(true, s.listDevices))
	m.HandleFunc("POST /api/devices", s.auth(true, s.addDevice))
	m.HandleFunc("DELETE /api/devices/{id}", s.auth(true, s.removeDevice))
	m.HandleFunc("GET /api/ha/state", s.deviceAuth(s.haState))
	m.HandleFunc("POST /api/ingest/event", s.deviceAuth(s.ingestEvent))
	m.HandleFunc("POST /api/ingest/points", s.deviceAuth(s.ingestPoints))
	m.HandleFunc("POST /api/ingest/owntracks", s.deviceAuth(s.ingestOwnTracks))
	m.HandleFunc("POST /api/track/points", s.auth(true, s.trackPoints))

	m.HandleFunc("GET /ford/connect", s.auth(true, s.fordConnect))
	m.HandleFunc("GET /ford/callback", s.callbackPage)
	m.HandleFunc("POST /api/ford/complete", s.auth(true, s.fordComplete))
	m.HandleFunc("POST /api/poll", s.auth(true, func(w http.ResponseWriter, r *http.Request) { s.P.Now(); ok(w, "queued") }))

	static, _ := fs.Sub(web.Files, "static")
	m.Handle("/", noCacheHTML(http.FileServer(http.FS(static))))
	return secHeaders(gzipHandler(m))
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
