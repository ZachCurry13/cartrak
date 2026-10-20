package fleet

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ZachCurry13/cartrak/internal/alerts"
	"github.com/ZachCurry13/cartrak/internal/config"
	"github.com/ZachCurry13/cartrak/internal/connector"
	"github.com/ZachCurry13/cartrak/internal/diskfree"
	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/poller"
	"github.com/ZachCurry13/cartrak/internal/push"
	"github.com/ZachCurry13/cartrak/internal/server"
	"github.com/ZachCurry13/cartrak/internal/settings"
	"github.com/ZachCurry13/cartrak/internal/store"
	"github.com/ZachCurry13/cartrak/internal/trips"
	"github.com/ZachCurry13/cartrak/internal/tunnel"
	"github.com/ZachCurry13/cartrak/internal/updater"
)

// MaxCars is how many cars one CarTrak follows: each one is polled from Ford on its own.
const MaxCars = 6

// Runtime is everything that belongs to one car.
type Runtime struct {
	Car    store.Car
	S      *store.Store
	Cfgs   *settings.Settings
	P      *poller.Poller
	Det    *trips.Detector
	Eng    *alerts.Engine
	Srv    *server.Server
	h      http.Handler // Srv.Handler(), built once
	cancel context.CancelFunc
}

// Fleet holds the core database and one Runtime per car.
type Fleet struct {
	Cfg     *config.Config
	Version string
	Core    *store.Store
	Upd     *updater.Checker
	Tun     *tunnel.Supervisor
	Conn    *connector.Manager
	Exit    func() // stops CarTrak so a confirmed restore can be applied at the next start

	mu    sync.RWMutex
	cars  map[int64]*Runtime
	first *store.Store // the first car's database, opened before the core so it can be copied from
	ctx   context.Context
}

// New opens the core database, making it from the first car's database on the first start (after a backup of that
// database), and prepares the shared pieces. Call Run to start the cars.
func New(cfg *config.Config, version string) (*Fleet, error) {
	f := &Fleet{Cfg: cfg, Version: version, cars: map[int64]*Runtime{}}
	first, err := store.Open(cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("database: %w", err)
	}
	f.first = first
	fresh := !store.CoreExists(cfg.DataDir)
	if fresh && first.CountUsers() > 0 {
		if err := backupBeforeMigration(first, cfg.DataDir); err != nil {
			return nil, fmt.Errorf("backup before the move to several cars: %w", err)
		}
	}
	if f.Core, err = store.OpenCore(cfg.DataDir); err != nil {
		return nil, fmt.Errorf("core database: %w", err)
	}
	if fresh {
		if err := store.ImportIdentity(f.Core, first, GlobalKeys()); err != nil {
			return nil, fmt.Errorf("copying the people to the core database: %w", err)
		}
	}
	if cars, _ := f.Core.ListCars(); len(cars) == 0 {
		if _, err := f.Core.AddCar("My car", "", 0); err != nil {
			return nil, err
		}
		f.GrantFirstCar()
	}
	f.Upd = updater.New(version)
	if u := os.Getenv("CARTRAK_UPDATE_URL"); u != "" { // testing aid: point at a fake releases endpoint
		f.Upd.URL = u
	}
	f.Tun = tunnel.New()
	if b := os.Getenv("CARTRAK_CLOUDFLARED"); b != "" { // optional: path to the cloudflared program
		f.Tun.Bin = b
	}
	f.Conn = &connector.Manager{Sup: f.Tun, Dir: filepath.Join(cfg.DataDir, "bin"), KV: f.Core}
	f.Conn.Restore() // use the connector installed from the app last time, before the tunnel starts
	return f, nil
}

// backupBeforeMigration keeps a copy of the first car's database from before the core database was made.
func backupBeforeMigration(st *store.Store, dataDir string) error {
	dir := filepath.Join(dataDir, "backups")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	dest := filepath.Join(dir, "before-several-cars-"+time.Now().Format("20060102-150405")+".db")
	if err := st.Backup(dest, false); err != nil {
		return err
	}
	log.Printf("several cars: a copy of the database from before the change is at %s", dest)
	return nil
}

// GrantFirstCar gives everyone who has no access to any car access to the first one: administrators manage it and
// everyone else reads it. It runs when the first car is made and after the first administrator is created.
func (f *Fleet) GrantFirstCar() {
	cars, _ := f.Core.ListCars()
	if len(cars) == 0 {
		return
	}
	users, _ := f.Core.ListUsers()
	for _, u := range users {
		if m, _ := f.Core.UserAccess(u.ID); len(m) > 0 {
			continue
		}
		level := store.AccessViewer
		if u.Role == "admin" {
			level = store.AccessOwner
		}
		_ = f.Core.SetAccess(u.ID, cars[0].ID, level)
	}
}

func (f *Fleet) carDir(c store.Car) string {
	if c.Dir == "" {
		return f.Cfg.DataDir
	}
	return filepath.Join(f.Cfg.DataDir, c.Dir)
}

// canSee says whether a person has been given access to the car (the administrator role alone is not enough: an
// administrator is not told about a car that is not theirs).
func (f *Fleet) canSee(userID, carID int64) bool { return f.Core.Access(userID, carID) != "" }

func (f *Fleet) provider(cfgs *settings.Settings, st *store.Store) (ford.Provider, server.Linker, server.Prober, server.RedirectChecker, error) {
	switch f.Cfg.Provider {
	case "fordconnect": // FordConnect 2.0 (current)
		c := ford.NewConnect2(func() ford.Connect2Opts {
			return ford.Connect2Opts{ClientID: cfgs.Value("FORD_CLIENT_ID"), ClientSecret: cfgs.Value("FORD_CLIENT_SECRET"),
				RedirectURL: cfgs.Value("FORD_REDIRECT_URL"), AuthURL: cfgs.Value("FORD_AUTH_URL")}
		}, st)
		return c, c, c, c, nil
	case "fordconnect1": // older FordConnect 1 API, kept for apps registered for it
		c := ford.NewConnectFn(func() ford.ConnectOpts {
			return ford.ConnectOpts{ClientID: cfgs.Value("FORD_CLIENT_ID"), ClientSecret: cfgs.Value("FORD_CLIENT_SECRET"),
				ApplicationID: cfgs.Value("FORD_APPLICATION_ID"), RedirectURL: cfgs.Value("FORD_REDIRECT_URL"), VehicleID: cfgs.Value("FORD_VIN")}
		}, st)
		return c, c, nil, nil, nil
	case "demo":
		return ford.NewDemoKind(os.Getenv("CARTRAK_DEMO_POWERTRAIN")), nil, nil, nil, nil // demo only: ev (default), gas, hev or phev
	}
	return nil, nil, nil, nil, fmt.Errorf("provider %q is not available (use fordconnect, fordconnect1 or demo)", f.Cfg.Provider)
}

// build makes the runtime of one car; nothing is started yet.
func (f *Fleet) build(c store.Car, primary bool) (*Runtime, error) {
	var st *store.Store
	if c.Dir == "" && f.first != nil {
		st = f.first
	} else {
		var err error
		if st, err = store.Open(f.carDir(c)); err != nil {
			return nil, fmt.Errorf("database of %q: %w", c.Name, err)
		}
	}
	cfgs := settings.New(&CarKV{Car: st, Core: f.Core, Primary: primary}, f.Cfg.Lookup)
	prov, link, prober, redirects, err := f.provider(cfgs, st)
	if err != nil {
		return nil, err
	}
	det := &trips.Detector{S: st, TZ: f.Cfg.TZ, TZFn: cfgs.Location}
	p := poller.New(prov, st, det, poller.Intervals{Driving: f.Cfg.PollDriving, Charging: f.Cfg.PollCharge, Parked: f.Cfg.PollParked, Live: f.Cfg.PollLive})
	srv := &server.Server{S: st, Core: f.Core, CarID: c.ID, P: p, D: det, Link: link, Version: f.Version, Upd: f.Upd, Cfg: cfgs, Prober: prober, Redirects: redirects,
		Tunnel: f.Tun, Conn: f.Conn, Port: f.Cfg.Addr[strings.LastIndex(f.Cfg.Addr, ":")+1:], TrustProxy: f.Cfg.TrustProxy, SessionDays: f.Cfg.SessionDays,
		TileDir: filepath.Join(f.Cfg.DataDir, "tiles"), DataDir: f.Cfg.DataDir, Exit: f.Exit}
	srv.Cars = f
	p.After = srv.Broadcast
	id := c.ID
	pushSvc := &push.Service{KV: f.Core, Subs: f.Core, Location: cfgs.Location,
		Visible: func(uid int64) bool { return f.canSee(uid, id) }, CarName: func() string { return f.labelFor(id) }}
	pushSvc.Log = func(a push.Alert, devices, wanted, quiet, sent int) {
		_ = st.LogAlert(store.AlertLogEntry{Kind: a.Kind, Title: a.Title, Body: a.Body, Devices: devices, Wanted: wanted, Quiet: quiet, Sent: sent})
	}
	srv.Push = pushSvc
	eng := &alerts.Engine{S: st, N: pushSvc, Async: true, ReminderTime: cfgs.PlugReminderTime, ReminderBelow: cfgs.PlugReminderBelow, BumpLevel: cfgs.BumpThreshold,
		LowBatteryLevel: cfgs.LowBatteryLevel, ChargeBudget: cfgs.ChargeBudget, MonthCost: srv.MonthChargingCost,
		DiskSpace: func() (uint64, uint64, bool) { return diskfree.Space(f.Cfg.DataDir) }, Loc: cfgs.Location,
		Link: func() (bool, int64) { s := p.Status(); return s.Linked, s.LastOK }}
	p.OnReading = eng.Handle
	return &Runtime{Car: c, S: st, Cfgs: cfgs, P: p, Det: det, Eng: eng, Srv: srv, h: srv.Handler()}, nil
}

// labelFor is the car's name for alerts, but only when there is more than one car (a single car needs no label).
func (f *Fleet) labelFor(id int64) string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if len(f.cars) < 2 {
		return ""
	}
	if r := f.cars[id]; r != nil {
		return r.Car.Name
	}
	return ""
}

// start runs one car's background work until ctx ends.
func (f *Fleet) start(ctx context.Context, r *Runtime) {
	ctx, r.cancel = context.WithCancel(ctx)
	go r.P.Run(ctx)
	go r.Srv.CleanupLoop(ctx)
	go r.Eng.Run(ctx)
	go func() { // copy the charted values of older readings into the small trends table, gently
		if err := r.S.BackfillTrends(ctx, 50, 500*time.Millisecond); err != nil && ctx.Err() == nil {
			log.Printf("trend backfill: %v", err)
		}
	}()
}

// Run builds and starts every car, then the shared services, and returns once they are running.
func (f *Fleet) Run(ctx context.Context) error {
	f.ctx = ctx
	list, err := f.Core.ListCars()
	if err != nil {
		return err
	}
	for i, c := range list {
		r, err := f.build(c, i == 0)
		if err != nil {
			return err
		}
		f.mu.Lock()
		f.cars[c.ID] = r
		f.mu.Unlock()
	}
	front := f.Front()
	if front == nil {
		return fmt.Errorf("no car to follow")
	}
	front.Srv.ApplyTunnel()
	before := f.Core.CountUsers()
	front.Srv.SeedAdmin(f.Cfg.AdminUser, f.Cfg.AdminPass, f.Cfg.ResetAdmin)
	if before == 0 {
		f.GrantFirstCar()
	}
	f.mu.RLock()
	for _, r := range f.cars {
		f.start(ctx, r)
	}
	f.mu.RUnlock()
	go f.Upd.Run(ctx, front.Srv.UpdateCheckEnabled)
	f.Conn.Enabled = front.Srv.UpdateCheckEnabled
	go f.Conn.Run(ctx)
	return nil
}

// Front is the first car's runtime, which answers for the whole installation (sign-in, people, settings of the site).
func (f *Fleet) Front() *Runtime {
	f.mu.RLock()
	defer f.mu.RUnlock()
	var best *Runtime
	for _, r := range f.cars {
		if best == nil || r.Car.ID < best.Car.ID {
			best = r
		}
	}
	return best
}

// Runtime returns one car's runtime, or nil.
func (f *Fleet) Runtime(id int64) *Runtime {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.cars[id]
}

// Close stops the shared services and closes every database.
func (f *Fleet) Close() {
	f.Tun.Stop()
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, r := range f.cars {
		if r.cancel != nil {
			r.cancel()
		}
		_ = r.S.Close()
	}
	_ = f.Core.Close()
}
