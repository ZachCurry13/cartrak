package fleet

import (
	"context"
	"fmt"
	"log"
	"time"
)

// start runs one car's background work until ctx ends.
func (f *Fleet) start(ctx context.Context, r *Runtime) {
	ctx, r.cancel = context.WithCancel(ctx)
	go r.P.Run(ctx)
	go r.Srv.CleanupLoop(ctx)
	go r.Srv.GasPricesRun(ctx)
	go r.Srv.ReasonsWarmup(ctx)
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
