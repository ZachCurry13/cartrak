// Package poller fetches vehicle state from Ford on an adaptive schedule.
package poller

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/store"
	"github.com/ZachCurry13/cartrak/internal/trips"
)

type Intervals struct{ Driving, Charging, Parked time.Duration }

// Status is what the UI shows about the connection to Ford.
type Status struct {
	Provider  string `json:"provider"`
	Linked    bool   `json:"linked"`
	LastOK    int64  `json:"lastOk"`
	LastError string `json:"lastError"`
	LastTry   int64  `json:"lastTry"`
	NextTry   int64  `json:"nextTry"`
}

type Poller struct {
	P    ford.Provider
	S    *store.Store
	D    *trips.Detector
	Iv   Intervals
	kick chan struct{}

	mu sync.Mutex
	st Status
}

func New(p ford.Provider, s *store.Store, d *trips.Detector, iv Intervals) *Poller {
	return &Poller{P: p, S: s, D: d, Iv: iv, kick: make(chan struct{}, 1), st: Status{Provider: p.Name()}}
}

func (p *Poller) Status() Status {
	p.mu.Lock()
	defer p.mu.Unlock()
	s := p.st
	s.Linked = p.P.Ready()
	return s
}

// Now asks for an immediate poll (e.g. right after linking the account).
func (p *Poller) Now() {
	select {
	case p.kick <- struct{}{}:
	default:
	}
}

// Run blocks until ctx is cancelled.
func (p *Poller) Run(ctx context.Context) {
	wait := time.Second
	for {
		p.set(func(s *Status) { s.NextTry = time.Now().Add(wait).Unix() })
		select {
		case <-ctx.Done():
			return
		case <-p.kick:
		case <-time.After(wait):
		}
		wait = p.once(ctx, wait)
	}
}

func (p *Poller) once(ctx context.Context, prev time.Duration) time.Duration {
	if !p.P.Ready() {
		return 15 * time.Second
	}
	now := time.Now()
	p.set(func(s *Status) { s.LastTry = now.Unix() })
	st, err := p.P.Fetch(ctx)
	if err != nil {
		if errors.Is(err, ford.ErrNotLinked) {
			return 15 * time.Second
		}
		log.Printf("poll failed: %v", err)
		p.set(func(s *Status) { s.LastError = err.Error() })
		return backoff(prev, p.Iv.Parked)
	}
	if _, err := p.S.SaveState(p.P.Name(), st); err != nil {
		log.Printf("save failed: %v", err)
	}
	if err := p.D.Process(st); err != nil {
		log.Printf("trip detect failed: %v", err)
	}
	p.set(func(s *Status) { s.LastOK, s.LastError = now.Unix(), "" })
	return p.Iv.pick(st)
}

func (iv Intervals) pick(st *ford.State) time.Duration {
	switch {
	case trips.Driving(st):
		return iv.Driving
	case st.ChargeKw != nil && *st.ChargeKw > 0.2:
		return iv.Charging
	}
	return iv.Parked
}

// backoff doubles the wait after each failure, capped at 30 minutes.
func backoff(prev, floor time.Duration) time.Duration {
	n := prev * 2
	if n < floor {
		n = floor
	}
	if n > 30*time.Minute {
		n = 30 * time.Minute
	}
	return n
}

func (p *Poller) set(f func(*Status)) {
	p.mu.Lock()
	f(&p.st)
	p.mu.Unlock()
}
