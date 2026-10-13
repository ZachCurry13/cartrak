package poller

import (
	"context"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
)

func TestFordServerErrorsAreRetriedSoonAndBackOffGently(t *testing.T) {
	boom := &ford.ServerError{Path: "/telemetry", Status: 500, Detail: "message: oops"}
	p := newPoller(t, scripted{err: boom}) // the parked pace is 10 minutes: a Ford outage must not wait that long
	want := []time.Duration{time.Minute, 2 * time.Minute, 4 * time.Minute, 8 * time.Minute, 10 * time.Minute, 10 * time.Minute}
	for i, w := range want {
		if got := p.once(context.Background(), 10*time.Minute); got != w {
			t.Fatalf("failure %d: waited %v, want %v", i+1, got, w)
		}
	}
	st := p.Status()
	if !st.FordTrouble || st.RateLimited || st.LastError != "ford /telemetry: HTTP 500 (Ford said: message: oops)" {
		t.Fatalf("status: %+v", st)
	}

	// Ford answers again: the flag clears and the next outage starts from a minute again.
	p.P = scripted{state: &ford.State{TS: time.Now(), Ignition: "OFF"}}
	p.once(context.Background(), time.Minute)
	if st := p.Status(); st.FordTrouble || st.LastError != "" {
		t.Fatalf("recovered but still flagged: %+v", st)
	}
	p.P = scripted{err: boom}
	if got := p.once(context.Background(), time.Minute); got != time.Minute {
		t.Fatalf("a new outage should start again at a minute, got %v", got)
	}
}

func TestOtherFailuresDoNotLookLikeFordBeingDown(t *testing.T) {
	p := newPoller(t, scripted{err: &ford.ServerError{Status: 502}})
	p.once(context.Background(), time.Second)
	p.P = scripted{err: context.DeadlineExceeded}
	p.once(context.Background(), time.Second)
	if st := p.Status(); st.FordTrouble || st.LastError == "" {
		t.Fatalf("a timeout is a plain failure: %+v", st)
	}
}
