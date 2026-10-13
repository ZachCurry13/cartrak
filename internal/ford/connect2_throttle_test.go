package ford

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestOneSlowEndpointPerReading(t *testing.T) {
	f := newFake2(t)
	c := f.client(memKV{refreshKey: "R0"})
	c.minGap = time.Millisecond // let every reading really ask Ford in this test

	s, err := c.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got := atomic.LoadInt32(&f.slowCalls); got != 1 {
		t.Fatalf("first reading should call exactly one slow endpoint, got %d", got)
	}
	if !strings.Contains(string(s.Raw), `"garage"`) || strings.Contains(string(s.Raw), `"chargeSchedules"`) {
		t.Fatalf("only the first slow endpoint should be present yet: %s", s.Raw)
	}
	for i := 0; i < 4; i++ {
		time.Sleep(3 * time.Millisecond)
		s, _ = c.Fetch(context.Background())
	}
	if got := atomic.LoadInt32(&f.slowCalls); got != 5 || !strings.Contains(string(s.Raw), `"chargeSchedules"`) {
		t.Fatalf("after five readings all five slow endpoints should be read once each: calls=%d raw=%s", got, s.Raw)
	}
	// Nothing is due again until slowEvery has passed.
	time.Sleep(3 * time.Millisecond)
	c.Fetch(context.Background())
	if got := atomic.LoadInt32(&f.slowCalls); got != 5 {
		t.Fatalf("slow endpoints re-read too soon: %d", got)
	}
	c.mu.Lock()
	for k := range c.slowAt {
		c.slowAt[k] = time.Now().Add(-2 * slowEvery)
	}
	c.mu.Unlock()
	time.Sleep(3 * time.Millisecond)
	c.Fetch(context.Background())
	if got := atomic.LoadInt32(&f.slowCalls); got != 6 {
		t.Fatalf("an overdue slow endpoint should be re-read (one per reading): %d", got)
	}
}

// Asking the same endpoint again within the gap serves the earlier answer instead of calling Ford.
func TestRecentAnswerIsServedFromMemory(t *testing.T) {
	f := newFake2(t)
	c := f.client(memKV{refreshKey: "R0"})
	for i := 0; i < 3; i++ {
		if _, err := c.Fetch(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if got := atomic.LoadInt32(&f.telemetryHit); got != 1 {
		t.Fatalf("Ford should have been asked once, not %d times, within the gap", got)
	}
}

// After a 429 nothing at all is sent until the wait is over (the limit may be shared).
func TestNothingIsSentWhileFordAsksUsToWait(t *testing.T) {
	f := newFake2(t)
	f.limited = true
	c := f.client(memKV{refreshKey: "R0"})
	if _, err := c.Fetch(context.Background()); err == nil {
		t.Fatal("expected a rate limit error")
	}
	hits := atomic.LoadInt32(&f.telemetryHit)
	var rl *RateLimitError
	_, err := c.Fetch(context.Background())
	if !errors.As(err, &rl) || rl.After <= 0 || atomic.LoadInt32(&f.telemetryHit) != hits {
		t.Fatalf("second attempt must not reach Ford while on hold: err=%v hits %d->%d", err, hits, f.telemetryHit)
	}
	// A different endpoint is held too.
	if _, status, err := c.call(context.Background(), "/garage"); status != 429 || !errors.As(err, &rl) || atomic.LoadInt32(&f.slowCalls) != 0 {
		t.Fatalf("other endpoints must be held as well: status=%d err=%v slow=%d", status, err, f.slowCalls)
	}
	// Once the hold is over, it asks again.
	c.mu.Lock()
	c.holdUntil = time.Now().Add(-time.Second)
	c.mu.Unlock()
	f.limited = false
	if _, err := c.Fetch(context.Background()); err != nil {
		t.Fatalf("should recover after the hold: %v", err)
	}
}

func TestProbeExplainsRateLimiting(t *testing.T) {
	f := newFake2(t)
	f.limited = true
	c := f.client(memKV{refreshKey: "R0"})
	c.Fetch(context.Background()) // gets the 429 and starts the hold
	res := c.Probe(context.Background())
	if res["telemetry"].Status != 429 || !strings.Contains(res["telemetry"].Note, "Wait about") {
		t.Fatalf("probe should say what to do on a 429: %+v", res["telemetry"])
	}
}

func TestSlowEndpointFailureIsNotFatal(t *testing.T) {
	f := newFake2(t)
	f.garageStatus = 403
	c := f.client(memKV{refreshKey: "R0"})
	s, err := c.Fetch(context.Background())
	if err != nil || s.SoCPct == nil {
		t.Fatalf("telemetry should still work: %v", err)
	}
	if !strings.Contains(string(s.Raw), `"_endpointErrors"`) || !strings.Contains(string(s.Raw), "HTTP 403") {
		t.Fatalf("failure not recorded: %s", s.Raw)
	}
}

func TestRateLimitV2(t *testing.T) {
	f := newFake2(t)
	f.limited = true
	c := f.client(memKV{refreshKey: "R0"})
	_, err := c.Fetch(context.Background())
	var rl *RateLimitError
	if !errors.As(err, &rl) || rl.After != 42*time.Second {
		t.Fatalf("want RateLimitError(42s), got %v", err)
	}
}
