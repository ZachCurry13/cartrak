// Command cartrak is a self-hosted trip and data tracker for a Ford EV.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
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

// version is set at build time (-ldflags -X main.version=...).
var version = "dev"

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	// A backup the owner confirmed in the app is swapped in now, before the database is opened.
	if done, err := store.ApplyPendingRestore(cfg.DataDir); err != nil {
		log.Printf("restore: %v", err)
	} else if done {
		log.Printf("restore: the backup was put in place; the previous data is kept next to it as cartrak.db.before-restore-*")
	}
	st, err := store.Open(cfg.DataDir)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer st.Close()

	cfgs := settings.New(st, cfg.Lookup)
	var prov ford.Provider
	var link server.Linker
	var prober server.Prober
	var redirects server.RedirectChecker
	switch cfg.Provider {
	case "fordconnect": // FordConnect 2.0 (current)
		c := ford.NewConnect2(func() ford.Connect2Opts {
			return ford.Connect2Opts{ClientID: cfgs.Value("FORD_CLIENT_ID"), ClientSecret: cfgs.Value("FORD_CLIENT_SECRET"),
				RedirectURL: cfgs.Value("FORD_REDIRECT_URL"), AuthURL: cfgs.Value("FORD_AUTH_URL")}
		}, st)
		prov, link, prober, redirects = c, c, c, c
	case "fordconnect1": // older FordConnect 1 API, kept for apps registered for it
		c := ford.NewConnectFn(func() ford.ConnectOpts {
			return ford.ConnectOpts{ClientID: cfgs.Value("FORD_CLIENT_ID"), ClientSecret: cfgs.Value("FORD_CLIENT_SECRET"),
				ApplicationID: cfgs.Value("FORD_APPLICATION_ID"), RedirectURL: cfgs.Value("FORD_REDIRECT_URL"), VehicleID: cfgs.Value("FORD_VIN")}
		}, st)
		prov, link = c, c
	case "demo":
		prov = ford.NewDemoKind(os.Getenv("CARTRAK_DEMO_POWERTRAIN")) // demo only: ev (default), gas, hev or phev
	default:
		log.Fatalf("provider %q is not available (use fordconnect, fordconnect1 or demo)", cfg.Provider)
	}

	det := &trips.Detector{S: st, TZ: cfg.TZ, TZFn: cfgs.Location}
	p := poller.New(prov, st, det, poller.Intervals{Driving: cfg.PollDriving, Charging: cfg.PollCharge, Parked: cfg.PollParked, Live: cfg.PollLive})
	upd := updater.New(version)
	if u := os.Getenv("CARTRAK_UPDATE_URL"); u != "" { // testing aid: point at a fake releases endpoint
		upd.URL = u
	}
	tun := tunnel.New()
	if b := os.Getenv("CARTRAK_CLOUDFLARED"); b != "" { // optional: path to the cloudflared program
		tun.Bin = b
	}
	conn := &connector.Manager{Sup: tun, Dir: filepath.Join(cfg.DataDir, "bin"), KV: st}
	conn.Restore() // use the connector installed from the app last time, before the tunnel starts
	port := cfg.Addr[strings.LastIndex(cfg.Addr, ":")+1:]
	srv := &server.Server{S: st, P: p, D: det, Link: link, Version: version, Upd: upd, Cfg: cfgs, Prober: prober, Redirects: redirects,
		Tunnel: tun, Conn: conn, Port: port, TrustProxy: cfg.TrustProxy, SessionDays: cfg.SessionDays,
		TileDir: filepath.Join(cfg.DataDir, "tiles"), DataDir: cfg.DataDir,
		Exit: func() { log.Printf("stopping so the restore can be applied"); st.Close(); os.Exit(1) }}
	p.After = srv.Broadcast
	pushSvc := &push.Service{KV: st, Subs: st, Location: cfgs.Location}
	pushSvc.Log = func(a push.Alert, devices, wanted, quiet, sent int) {
		_ = st.LogAlert(store.AlertLogEntry{Kind: a.Kind, Title: a.Title, Body: a.Body, Devices: devices, Wanted: wanted, Quiet: quiet, Sent: sent})
	}
	srv.Push = pushSvc
	alertEngine := &alerts.Engine{S: st, N: pushSvc, Async: true, ReminderTime: cfgs.PlugReminderTime, ReminderBelow: cfgs.PlugReminderBelow, BumpLevel: cfgs.BumpThreshold, DiskSpace: func() (uint64, uint64, bool) { return diskfree.Space(cfg.DataDir) }, Loc: cfgs.Location, Link: func() (bool, int64) { s := p.Status(); return s.Linked, s.LastOK }}
	p.OnReading = alertEngine.Handle
	srv.ApplyTunnel()
	defer tun.Stop()
	srv.SeedAdmin(cfg.AdminUser, cfg.AdminPass, cfg.ResetAdmin)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go p.Run(ctx)
	go upd.Run(ctx, srv.UpdateCheckEnabled)
	conn.Enabled = srv.UpdateCheckEnabled
	go conn.Run(ctx)
	go srv.CleanupLoop(ctx)
	go func() { // copy the charted values of older readings into the small trends table, gently
		if err := st.BackfillTrends(ctx, 50, 500*time.Millisecond); err != nil && ctx.Err() == nil {
			log.Printf("trend backfill: %v", err)
		}
	}()
	go alertEngine.Run(ctx)

	hs := &http.Server{Addr: cfg.Addr, Handler: srv.Handler(), ReadHeaderTimeout: 10 * time.Second}
	go func() {
		<-ctx.Done()
		sc, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		tun.Stop()
		_ = hs.Shutdown(sc)
	}()
	log.Printf("cartrak %s listening on %s (provider %s)", version, cfg.Addr, prov.Name())
	if err := hs.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
