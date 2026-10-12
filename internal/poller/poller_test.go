package poller

import (
	"context"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/store"
	"github.com/ZachCurry13/cartrak/internal/trips"
)

type scripted struct {
	err   error
	state *ford.State
}

func (scripted) Name() string { return "scripted" }
func (scripted) Ready() bool  { return true }
func (s scripted) Fetch(context.Context) (*ford.State, error) {
	return s.state, s.err
}

func newPoller(t *testing.T, p ford.Provider) *Poller {
	s, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return New(p, s, &trips.Detector{S: s, TZ: time.UTC},
		Intervals{Driving: 30 * time.Second, Charging: 2 * time.Minute, Parked: 10 * time.Minute, Live: 10 * time.Second})
}

func TestRateLimitIsObeyed(t *testing.T) {
	p := newPoller(t, scripted{err: &ford.RateLimitError{After: 90 * time.Second}})
	if w := p.once(context.Background(), time.Second); w != 90*time.Second {
		t.Fatalf("should wait what Ford asked: %v", w)
	}
	if st := p.Status(); !st.RateLimited || st.LastError == "" {
		t.Fatalf("status not showing the slowdown: %+v", st)
	}
	// Ford gave no hint (or a tiny one): never hammer it.
	p2 := newPoller(t, scripted{err: &ford.RateLimitError{}})
	if w := p2.once(context.Background(), time.Second); w != minRateLimited {
		t.Fatalf("default wait: %v", w)
	}
}

func TestLiveModeSpeedsUpOnlyWhileDriving(t *testing.T) {
	p := newPoller(t, scripted{})
	driving := &ford.State{Ignition: "ON"}
	parked := &ford.State{Ignition: "OFF"}

	if got := p.interval(driving); got != 30*time.Second {
		t.Fatalf("normal driving: %v", got)
	}
	p.SetLive(true)
	if got := p.interval(driving); got != 10*time.Second {
		t.Fatalf("live driving: %v", got)
	}
	if got := p.interval(parked); got != 10*time.Minute {
		t.Fatalf("live mode must not speed up parked polling: %v", got)
	}
	st := p.Status()
	if !st.Live || st.LiveUntil == 0 {
		t.Fatalf("status: %+v", st)
	}
	p.SetLive(false)
	if got := p.interval(driving); got != 30*time.Second || p.Status().Live {
		t.Fatalf("live mode did not turn off: %v", got)
	}
	// A silly setting cannot go below the safety floor.
	p.Iv.Live = time.Second
	p.SetLive(true)
	if got := p.interval(driving); got != minLive {
		t.Fatalf("floor not applied: %v", got)
	}
}

func TestSuccessfulPollRunsAfterHook(t *testing.T) {
	soc := 60.0
	p := newPoller(t, scripted{state: &ford.State{TS: time.Now(), Ignition: "OFF", SoCPct: &soc, Raw: []byte(`{}`)}})
	called := 0
	p.After = func() { called++ }
	p.once(context.Background(), time.Second)
	if called != 1 || p.Status().LastOK == 0 {
		t.Fatalf("after hook calls=%d status=%+v", called, p.Status())
	}
}
