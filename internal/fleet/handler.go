package fleet

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/ZachCurry13/cartrak/internal/server"
	"github.com/ZachCurry13/cartrak/internal/store"
)

// frontPaths are answered by the first car's server whichever car is chosen: signing in, the people, notification
// devices and the whole installation's settings. They use only the core database, and keeping them on one server keeps
// one count of wrong passwords.
var frontPaths = []string{"/api/login", "/api/setup", "/api/logout", "/api/me", "/api/users", "/api/security",
	"/api/push/", "/api/cars", "/api/inbox", "/api/connector", "/api/backup/site", "/api/changelog", "/api/settings/update-check", "/api/admin/remote", "/api/site/", "/healthz"}

func isFront(path string) bool {
	for _, p := range frontPaths {
		if path == p || strings.HasPrefix(path, p) && (strings.HasSuffix(p, "/") || path[len(p)] == '/') {
			return true
		}
	}
	return false
}

// Handler sends each request to the server of the right car.
func (f *Fleet) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rt := f.pick(r)
		if rt == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"no such car"}`))
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") && !isFront(r.URL.Path) && asked(r) != "" { // a car's own answer says which car it is, so the app can tell when it is not the one it asked about
			w.Header().Set("X-Car-Id", strconv.FormatInt(rt.Car.ID, 10))
		}
		rt.h.ServeHTTP(w, r)
	})
}

// pick finds the car a request is about, falling back to the first car so something always answers. A car the request
// asks for by number that does not exist (any more) gives nil, which is answered with 404 rather than another car's data.
func (f *Fleet) pick(r *http.Request) *Runtime {
	p := r.URL.Path
	if isFront(p) {
		return f.Front()
	}
	switch {
	case strings.HasPrefix(p, "/api/ingest/") || p == "/api/ha/state":
		if tok := server.DeviceToken(r); tok != "" {
			for _, rt := range f.all() {
				if rt.Srv.HasDevice(tok) {
					return rt
				}
			}
		}
		return f.Front()
	case strings.HasPrefix(p, "/s/"):
		tok, _, _ := strings.Cut(strings.TrimPrefix(p, "/s/"), "/")
		for _, rt := range f.all() {
			if rt.Srv.HasShare(tok) {
				return rt
			}
		}
		return f.Front()
	case p == "/ford/callback":
		if rt := f.Runtime(server.CarFromState(r.URL.Query().Get("state"))); rt != nil {
			return rt
		}
	}
	if asked := asked(r); asked != "" {
		id, _ := strconv.ParseInt(asked, 10, 64)
		return f.Runtime(id)
	}
	if rt := f.chosen(r); rt != nil {
		return rt
	}
	if rt := f.defaultFor(r); rt != nil {
		return rt
	}
	return f.Front()
}

// asked is the car number a request names itself: ?car=2 or the X-Car header (the app sets it on every call).
func asked(r *http.Request) string {
	if v := r.URL.Query().Get("car"); v != "" {
		return v
	}
	return r.Header.Get("X-Car")
}

// chosen is the car a browser asked for: ?car=2, the X-Car header (the app sets it on every call), or the car it looked
// at last.
func (f *Fleet) chosen(r *http.Request) *Runtime {
	v := r.URL.Query().Get("car")
	if v == "" {
		v = r.Header.Get("X-Car")
	}
	if v == "" {
		if c, err := r.Cookie(server.CarCookie); err == nil {
			v = c.Value
		}
	}
	id, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return nil
	}
	return f.Runtime(id)
}

// defaultFor is the first car the signed-in person has access to.
func (f *Fleet) defaultFor(r *http.Request) *Runtime {
	c, err := r.Cookie(server.CookieName)
	if err != nil {
		return nil
	}
	u, _ := f.Core.SessionUser(c.Value)
	if u == nil {
		return nil
	}
	cars, _ := f.Core.ListCars()
	access, _ := f.Core.UserAccess(u.ID)
	for _, car := range cars {
		if _, ok := access[car.ID]; ok {
			return f.Runtime(car.ID)
		}
	}
	if u.Role == "admin" && len(cars) > 0 {
		return f.Runtime(cars[0].ID)
	}
	return nil
}

func (f *Fleet) all() []*Runtime {
	f.mu.RLock()
	defer f.mu.RUnlock()
	out := make([]*Runtime, 0, len(f.cars))
	for _, r := range f.cars {
		out = append(out, r)
	}
	return out
}

var _ = store.AccessOwner
