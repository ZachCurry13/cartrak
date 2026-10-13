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
		Intervals{Driving: 90 * time.Second, Charging: 2 * time.Minute, Parked: 10 * time.Minute, Live: 45 * time.Second})
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

	if got := p.interval(driving); got != 90*time.Second {
		t.Fatalf("normal driving: %v", got)
	}
	p.SetLive(true)
	if got := p.interval(driving); got != 45*time.Second {
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
	if got := p.interval(driving); got != 90*time.Second || p.Status().Live {
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

func TestRateLimitBackoffDoublesAndIsCapped(t *testing.T) {
	cases := []struct {
		asked time.Duration
		n     int
		want  time.Duration
	}{
		{0, 1, 30 * time.Second}, {0, 2, time.Minute}, {0, 3, 2 * time.Minute}, {0, 4, 4 * time.Minute},
		{0, 5, 8 * time.Minute}, {0, 6, 10 * time.Minute}, {0, 20, 10 * time.Minute},
		{5 * time.Second, 1, 30 * time.Second}, // never below the floor
		{3 * time.Minute, 1, 3 * time.Minute},  // obey a longer request from Ford
		{time.Hour, 1, 10 * time.Minute},       // but never wait forever
	}
	for _, c := range cases {
		if got := rateLimitWait(c.asked, c.n); got != c.want {
			t.Errorf("rateLimitWait(%v, %d) = %v, want %v", c.asked, c.n, got, c.want)
		}
	}
}

func TestRepeatedRateLimitsBackOffMoreThenRecover(t *testing.T) {
	rl := &ford.RateLimitError{}
	pr := &flaky{err: rl}
	p := newPoller(t, pr)
	w1 := p.once(context.Background(), time.Second)
	w2 := p.once(context.Background(), time.Second)
	if w1 != 30*time.Second || w2 != time.Minute {
		t.Fatalf("waits should double: %v then %v", w1, w2)
	}
	soc := 50.0
	pr.err, pr.state = nil, &ford.State{TS: time.Now(), Ignition: "OFF", SoCPct: &soc, Raw: []byte(`{}`)}
	p.once(context.Background(), time.Second)
	pr.err = rl
	if w := p.once(context.Background(), time.Second); w != 30*time.Second {
		t.Fatalf("a success must reset the backoff: %v", w)
	}
}

type flaky struct {
	err   error
	state *ford.State
}

func (*flaky) Name() string { return "flaky" }
func (*flaky) Ready() bool  { return true }
func (f *flaky) Fetch(context.Context) (*ford.State, error) {
	return f.state, f.err
}
