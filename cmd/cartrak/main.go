// Command cartrak is a self-hosted trip and data tracker for a Ford EV.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ZachCurry13/cartrak/internal/config"
	"github.com/ZachCurry13/cartrak/internal/fleet"
	"github.com/ZachCurry13/cartrak/internal/store"
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
	f, err := fleet.New(cfg, version)
	if err != nil {
		log.Fatalf("start: %v", err)
	}
	defer f.Close()
	f.Exit = func() { log.Printf("stopping so the restore can be applied"); f.Close(); os.Exit(1) }

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := f.Run(ctx); err != nil {
		log.Fatalf("start: %v", err)
	}

	hs := &http.Server{Addr: cfg.Addr, Handler: f.Handler(), ReadHeaderTimeout: 10 * time.Second}
	go func() {
		<-ctx.Done()
		sc, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		f.Tun.Stop()
		_ = hs.Shutdown(sc)
	}()
	log.Printf("cartrak %s listening on %s (provider %s)", version, cfg.Addr, cfg.Provider)
	if err := hs.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
