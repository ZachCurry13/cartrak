// Package poller fetches vehicle state from Ford on an adaptive schedule.
package poller

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/store"
	"github.com/ZachCurry13/cartrak/internal/trips"
)

// Intervals are the waits between readings. Live applies only while driving
// and only while road-trip (live) mode is on.
type Intervals struct{ Driving, Charging, Parked, Live time.Duration }

const (
	minLive        = 30 * time.Second // Ford allows about one request per endpoint per 30 s, so faster is pointless
	minRateLimited = 30 * time.Second
	maxRateLimited = 10 * time.Minute
	liveMaxHours   = 12 // live mode switches itself off so it is never left on by accident
)

// Status is what the UI shows about the connection to Ford.
type Status struct {
	Provider    string `json:"provider"`
	Linked      bool   `json:"linked"`
	LastOK      int64  `json:"lastOk"`
	LastError   string `json:"lastError"`
	LastTry     int64  `json:"lastTry"`
	NextTry     int64  `json:"nextTry"`
	Live        bool   `json:"live"`
	LiveUntil   int64  `json:"liveUntil"`
	RateLimited bool   `json:"rateLimited"`
	FordTrouble bool   `json:"fordTrouble"` // Ford answered with a server error (HTTP 5xx); not a setup problem
}

type Poller struct {
	P  ford.Provider
	S  *store.Store
	D  *trips.Detector
	Iv Intervals
	// After runs after every reading that was saved (used to push live updates).
	After func()
	// OnReading runs with every reading that was saved and processed (used for alerts).
	OnReading func(*ford.State)
	kick      chan struct{}

	mu         sync.Mutex
	st         Status
	liveUntil  time.Time
	hintUntil  time.Time // a phone said a trip is under way until then
	chargeSeen time.Time // the last time a reading showed the car taking charge
	polls      []PollRec // the latest polls, for the Debug report (in memory: starts again at a restart)
	rlCount    int       // consecutive 429s, for the backoff
	srvCount   int       // consecutive Ford server errors (HTTP 5xx), for the retry pace
}

func New(p ford.Provider, s *store.Store, d *trips.Detector, iv Intervals) *Poller {
	return &Poller{P: p, S: s, D: d, Iv: iv, kick: make(chan struct{}, 1), st: Status{Provider: p.Name()}}
}

func (p *Poller) Status() Status {
	p.mu.Lock()
	defer p.mu.Unlock()
	s := p.st
	s.Linked = p.P.Ready()
	if time.Now().Before(p.liveUntil) {
		s.Live, s.LiveUntil = true, p.liveUntil.Unix()
	}
	return s
}

// SetLive turns road-trip mode on (for up to 12 hours) or off, and polls now.
func (p *Poller) SetLive(on bool) {
	p.mu.Lock()
	if on {
		p.liveUntil = time.Now().Add(liveMaxHours * time.Hour)
	} else {
		p.liveUntil = time.Time{}
	}
	p.mu.Unlock()
	p.Now()
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
		var rl *ford.RateLimitError
		if errors.As(err, &rl) { // Ford asked us to slow down: obey it
			p.mu.Lock()
			p.rlCount++
			w := rateLimitWait(rl.After, p.rlCount)
			p.mu.Unlock()
			log.Printf("ford rate limit: waiting %s", w)
			p.record(PollRec{TS: now.Unix(), Result: "rate limit", NextSec: int(w.Seconds()), Detail: "Ford asked CarTrak to slow down"})
			p.set(func(s *Status) {
				s.RateLimited, s.FordTrouble = true, false
				s.LastError = fmt.Sprintf("Ford asked CarTrak to slow down; trying again in %d s", int(w.Seconds()))
			})
			return w
		}
		var se *ford.ServerError
		if errors.As(err, &se) { // Ford answered HTTP 5xx: its side, so try again soon
			p.mu.Lock()
			p.srvCount++
			w := serverErrorWait(p.srvCount)
			p.mu.Unlock()
			log.Printf("ford is having trouble: %v; trying again in %s", err, w)
			p.record(PollRec{TS: now.Unix(), Result: "Ford error", NextSec: int(w.Seconds()), Detail: err.Error()})
			p.set(func(s *Status) { s.LastError, s.RateLimited, s.FordTrouble = err.Error(), false, true })
			return w
		}
		log.Printf("poll failed: %v", err)
		p.record(PollRec{TS: now.Unix(), Result: "failed", NextSec: int(backoff(prev, p.Iv.Parked).Seconds()), Detail: err.Error()})
		p.mu.Lock()
		p.srvCount = 0
		p.mu.Unlock()
		p.set(func(s *Status) { s.LastError, s.RateLimited, s.FordTrouble = err.Error(), false, false })
		return backoff(prev, p.Iv.Parked)
	}
	if _, err := p.S.SaveState(p.P.Name(), st); err != nil {
		log.Printf("save failed: %v", err)
	}
	p.LearnCaps(st)
	if err := p.D.Process(st); err != nil {
		log.Printf("trip detect failed: %v", err)
	}
	p.mu.Lock()
	p.rlCount, p.srvCount = 0, 0
	p.mu.Unlock()
	p.set(func(s *Status) { s.LastOK, s.LastError, s.RateLimited, s.FordTrouble = now.Unix(), "", false, false })
	if p.OnReading != nil { // after the status is updated, so alert rules see this reading as the latest
		p.OnReading(st)
	}
	if p.After != nil {
		p.After()
	}
	next := p.interval(st)
	p.record(PollRec{TS: now.Unix(), Result: "ok", NextSec: int(next.Seconds()), Detail: fmt.Sprintf("ignition %s, plug %s, charge %s", st.Ignition, st.Plug, st.ChargeState)})
	return next
}

func (p *Poller) set(f func(*Status)) {
	p.mu.Lock()
	f(&p.st)
	p.mu.Unlock()
}
