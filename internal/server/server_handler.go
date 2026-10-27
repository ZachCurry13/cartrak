package server

import (
	"io/fs"
	"net/http"

	"github.com/ZachCurry13/cartrak/internal/web"
)

func (s *Server) Handler() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })

	m.HandleFunc("GET /api/setup", s.setupState)
	m.HandleFunc("POST /api/setup", s.setup)
	m.HandleFunc("POST /api/login", s.login)
	m.HandleFunc("POST /api/logout", s.authSite(false, s.logout))
	m.HandleFunc("GET /api/me", s.authSite(false, s.me))

	m.HandleFunc("GET /api/admin/settings", s.auth(true, s.getSettings))
	m.HandleFunc("PUT /api/admin/settings", s.auth(true, s.forgetSavings(s.saveSettings)))
	m.HandleFunc("POST /api/admin/ford-probe", s.auth(true, s.fordProbe))
	m.HandleFunc("POST /api/admin/ford/detect-redirect", s.auth(true, s.detectRedirect))
	m.HandleFunc("GET /api/admin/remote", s.authSite(true, s.getRemote))
	m.HandleFunc("PUT /api/admin/remote", s.authSite(true, s.saveRemote))
	m.HandleFunc("GET /api/about", s.auth(false, s.about))
	m.HandleFunc("POST /api/settings/update-check", s.authSite(true, s.setUpdateCheck))
	m.HandleFunc("GET /api/changelog", s.auth(false, s.changelog))
	m.HandleFunc("GET /api/onboarding", s.auth(false, s.onboarding))
	m.HandleFunc("POST /api/onboarding/dismiss", s.auth(true, s.dismissOnboarding))
	m.HandleFunc("GET /api/checklist", s.auth(false, s.setupChecklist))
	m.HandleFunc("GET /api/me/layout", s.auth(false, s.getLayout))
	m.HandleFunc("PUT /api/me/layout", s.auth(false, s.putLayout))
	m.HandleFunc("GET /api/fuel", s.auth(false, s.fuelPage))
	m.HandleFunc("GET /api/analytics/fuel", s.auth(false, s.fuelStats))
	m.HandleFunc("GET /api/analytics/places", s.auth(false, s.popularPlaces))
	m.HandleFunc("POST /api/fuel/fills", s.auth(true, s.addFill))
	m.HandleFunc("PATCH /api/fuel/fills/{id}", s.auth(true, s.patchFill))
	m.HandleFunc("DELETE /api/fuel/fills/{id}", s.auth(true, s.deleteFill))
	m.HandleFunc("GET /api/me/prefs", s.auth(false, s.getPrefs))
	m.HandleFunc("PUT /api/me/prefs", s.auth(false, s.putPrefs))
	m.HandleFunc("POST /api/checklist/hide", s.auth(false, s.hideSetup))
	m.HandleFunc("POST /api/checklist/skip", s.auth(false, s.skipSetup))

	s.dataRoutes(m)

	m.HandleFunc("GET /api/rules", s.auth(false, s.listRules))
	m.HandleFunc("POST /api/rules", s.auth(true, s.saveRule))
	m.HandleFunc("PUT /api/rules/{id}", s.auth(true, s.saveRule))
	m.HandleFunc("DELETE /api/rules/{id}", s.auth(true, s.deleteRule))
	m.HandleFunc("POST /api/rules/apply", s.auth(true, s.applyRules))

	m.HandleFunc("GET /api/inbox", s.authSite(false, s.inbox))
	m.HandleFunc("GET /api/cars", s.authSite(false, s.listCars))
	m.HandleFunc("POST /api/cars", s.authSite(false, s.addCar))
	m.HandleFunc("PATCH /api/cars/{id}", s.authSite(false, s.patchCar))
	m.HandleFunc("DELETE /api/cars/{id}", s.authSite(false, s.removeCar))
	m.HandleFunc("GET /api/cars/{id}/access", s.authSite(false, s.carAccess))
	m.HandleFunc("PUT /api/cars/{id}/access", s.authSite(false, s.setCarAccess))
	m.HandleFunc("PATCH /api/users/{id}", s.authSite(true, s.patchUser))
	m.HandleFunc("GET /api/users", s.authSite(true, s.listUsers))
	m.HandleFunc("POST /api/users", s.authSite(true, s.addUser))
	m.HandleFunc("DELETE /api/users/{id}", s.authSite(true, s.deleteUser))

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
	m.HandleFunc("GET /api/ford/vehicles", s.auth(true, s.fordVehicles))
	m.HandleFunc("POST /api/ford/vehicle", s.auth(true, s.chooseVehicle))
	m.HandleFunc("POST /api/poll", s.auth(true, func(w http.ResponseWriter, r *http.Request) { s.P.Now(); ok(w, "queued") }))

	static, _ := fs.Sub(web.Files, "static")
	m.Handle("/", s.staticHandler(static))
	return recoverer(timed(secHeaders(gzipHandler(m))))
}
