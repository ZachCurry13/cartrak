// Package server exposes the JSON API and the embedded mobile web app.
package server

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ZachCurry13/cartrak/internal/connector"
	"github.com/ZachCurry13/cartrak/internal/poller"
	"github.com/ZachCurry13/cartrak/internal/push"
	"github.com/ZachCurry13/cartrak/internal/routeest"
	"github.com/ZachCurry13/cartrak/internal/settings"
	"github.com/ZachCurry13/cartrak/internal/store"
	"github.com/ZachCurry13/cartrak/internal/trips"
	"github.com/ZachCurry13/cartrak/internal/tunnel"
	"github.com/ZachCurry13/cartrak/internal/updater"
	"github.com/ZachCurry13/cartrak/internal/weather"
)

// Linker is implemented by providers that need the owner to sign in at Ford.
type Linker interface {
	LinkURL(state string) string
	Exchange(ctx context.Context, code string) error
}

type Server struct {
	S            *store.Store
	Core         *store.Store // the people, sign-ins and the list of cars; nil means this one database holds them (single car, tests)
	CarID        int64        // which car this server answers for; 0 when there is only one
	P            *poller.Poller
	D            *trips.Detector
	Link         Linker // nil when the provider needs no account link
	Version      string
	Cfg          *settings.Settings // admin-panel settings (nil in some tests)
	Prober       Prober             // optional: Ford diagnostics
	Redirects    RedirectChecker    // optional: ask Ford which redirect addresses are registered
	Tunnel       *tunnel.Supervisor // in-app Cloudflare connector (nil in some tests)
	Conn         *connector.Manager // keeps the connector up to date (nil in some tests)
	Port         string             // port the app listens on, for the setup steps
	hub          hub
	RouteEst     *routeest.Client // optional, for tests
	routeMu      sync.Mutex
	routeFail    map[int64]routeFailure // trips whose road estimate failed lately
	routeBusy    atomic.Bool            // background road estimates are running
	routeRetried atomic.Bool            // old give-ups were forgotten once this run
	tracking     trackSession           // the tracking session started on the Developer page
	effKept      effCache               // the km per kWh figure the savings use, kept for a minute
	savingsKept  savingsCache           // the savings page answer, kept briefly
	TrustProxy   bool                   // believe CF-Connecting-IP / X-Real-IP / X-Forwarded-For
	SessionDays  int                    // "keep me signed in" length; 0 = 30
	Upd          *updater.Checker       // nil disables update checks
	Weather      *weather.Client        // the optional wind lookup (nil uses the real service)
	lim          limiter
	devRate      rateWindow    // per-device call limit for the phone ingest API
	TileDir      string        // where downloaded map tiles are kept; empty switches maps off
	DataDir      string        // the data folder: backups are written here while they are made
	Exit         func()        // stops CarTrak so a confirmed restore can be applied at the next start (nil in tests)
	TileClient   *http.Client  // optional, for tests
	Push         *push.Service // Web Push alerts (nil in some tests)
	GasURL       string        // live gas price service; empty uses the EIA (tests point it at a fake)
	GeoGap       time.Duration // spacing of city lookups; 0 uses the service default (tests shorten it)
	geo          geoState
	gasLive      gasLiveState
	CityLookups  bool        // the trip list may look up the cities of new places in the background (on in the running app, off in tests)
	cityBusy     atomic.Bool // such a lookup is running
	tiles        tileFlight
	trim         trimState
	Cars         CarManager // adds, renames and removes cars (nil with a single car)
	cc           carCaches  // answers kept for a while, for this car only
}

// core is the database of people and sign-ins: the shared core database, or the one database when there is a single car.
func (s *Server) core() *store.Store {
	if s.Core != nil {
		return s.Core
	}
	return s.S
}
