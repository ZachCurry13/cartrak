package fleet

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ZachCurry13/cartrak/internal/alerts"
	"github.com/ZachCurry13/cartrak/internal/config"
	"github.com/ZachCurry13/cartrak/internal/connector"
	"github.com/ZachCurry13/cartrak/internal/poller"
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
		if err := first.ClearIdentity(); err != nil { // the old copy would only go stale, and it holds password hashes
			log.Printf("several cars: could not clear the old copy of the accounts: %v", err)
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

// start runs one car's background work until ctx ends.
func (f *Fleet) start(ctx context.Context, r *Runtime) {
	ctx, r.cancel = context.WithCancel(ctx)
	go r.P.Run(ctx)
	go r.Srv.CleanupLoop(ctx)
	go r.Eng.Run(ctx)
	upTo := r.S.LatestSampleID() // readings after this are seen live for remote starts
	go func() {                  // copy the charted values of older readings into the small trends table, gently, then find old remote starts
		if err := r.S.BackfillTrends(ctx, 50, 500*time.Millisecond); err != nil && ctx.Err() == nil {
			log.Printf("trend backfill: %v", err)
		}
		if err := r.S.BackfillRemoteStarts(ctx, upTo, 50, 500*time.Millisecond); err != nil && ctx.Err() == nil {
			log.Printf("remote start backfill: %v", err)
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
	go f.birthdayLoop(ctx)
	return nil
}
