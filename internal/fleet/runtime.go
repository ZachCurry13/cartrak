package fleet

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/ZachCurry13/cartrak/internal/alerts"
	"github.com/ZachCurry13/cartrak/internal/diskfree"
	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/poller"
	"github.com/ZachCurry13/cartrak/internal/push"
	"github.com/ZachCurry13/cartrak/internal/server"
	"github.com/ZachCurry13/cartrak/internal/settings"
	"github.com/ZachCurry13/cartrak/internal/store"
	"github.com/ZachCurry13/cartrak/internal/trips"
)

func (f *Fleet) provider(cfgs *settings.Settings, st *store.Store) (ford.Provider, server.Linker, server.Prober, server.RedirectChecker, error) {
	switch f.Cfg.Provider {
	case "fordconnect": // FordConnect 2.0 (current)
		c := ford.NewConnect2(func() ford.Connect2Opts {
			return ford.Connect2Opts{ClientID: cfgs.Value("FORD_CLIENT_ID"), ClientSecret: cfgs.Value("FORD_CLIENT_SECRET"),
				RedirectURL: cfgs.Value("FORD_REDIRECT_URL"), AuthURL: cfgs.Value("FORD_AUTH_URL"), VIN: cfgs.Value("FORD_VIN"), Taken: func() []string { return f.takenVINs(st) }}
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
		dir := f.carDir(c)
		if done, rerr := store.ApplyPendingRestoreTo(filepath.Join(dir, "restore"), filepath.Join(dir, "cartrak.db")); rerr != nil {
			log.Printf("restore for %q: %v", c.Name, rerr)
		} else if done {
			log.Printf("restore: the backup for %q was put in place; the previous data is kept next to it", c.Name)
		}
		if st, err = store.Open(dir); err != nil {
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
		TileDir: filepath.Join(f.Cfg.DataDir, "tiles"), DataDir: f.Cfg.DataDir, Exit: f.Exit, CityLookups: true}
	srv.Cars = f
	p.After = srv.Broadcast
	id := c.ID
	pushSvc := &push.Service{KV: f.Core, Subs: f.Core, Location: cfgs.Location,
		Visible: func(uid int64) bool { return f.canSee(uid, id) }, CarName: func() string { return f.labelFor(id) }, Prefs: f.Core.Prefs}
	pushSvc.Log = func(a push.Alert, devices, wanted, quiet, sent int) {
		_ = st.LogAlert(store.AlertLogEntry{Kind: a.Kind, Title: a.Title, Body: a.Body, Devices: devices, Wanted: wanted, Quiet: quiet, Sent: sent, URL: a.URL})
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
