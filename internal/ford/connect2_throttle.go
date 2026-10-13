package ford

import (
	"context"
	"errors"
	"time"
)

// Ford limits how often an app may ask: users of this API report that a second
// request to the same endpoint within a few seconds gets HTTP 429, and that waiting
// about 30 seconds avoids it. CarTrak therefore never asks the same endpoint more
// than once per defaultGap, serves a recent answer from memory instead, and stops
// sending anything while Ford has asked it to wait.
const defaultGap = 30 * time.Second

type cachedAnswer struct {
	body   []byte
	status int
	at     time.Time
}

func (c *Connect2) gap() time.Duration {
	if c.minGap > 0 {
		return c.minGap
	}
	return defaultGap
}

// call asks Ford for one endpoint, respecting the limits above.
func (c *Connect2) call(ctx context.Context, path string) ([]byte, int, error) {
	c.mu.Lock()
	if e, ok := c.cache[path]; ok && time.Since(e.at) < c.gap() {
		c.mu.Unlock()
		return e.body, e.status, nil
	}
	if wait := time.Until(c.holdUntil); wait > 0 {
		c.mu.Unlock()
		return nil, 429, &RateLimitError{After: wait}
	}
	c.mu.Unlock()

	b, status, err := c.callRaw(ctx, path)

	c.mu.Lock()
	defer c.mu.Unlock()
	var rl *RateLimitError
	if errors.As(err, &rl) {
		wait := rl.After
		if wait < c.gap() {
			wait = c.gap()
		}
		c.holdUntil = time.Now().Add(wait) // hold every endpoint, in case the limit is shared
		rl.After = wait
		return b, 429, rl
	}
	if err == nil && status == 200 {
		c.cache[path] = cachedAnswer{body: b, status: status, at: time.Now()}
	}
	return b, status, err
}

type slowEndpoint = struct{ Key, Path string }

// nextSlow returns the slow endpoint that is most overdue for a re-read, if any is due.
func (c *Connect2) nextSlow() (slowEndpoint, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var best slowEndpoint
	var bestAt time.Time
	found := false
	for _, e := range slowEndpoints {
		at := c.slowAt[e.Key] // zero if never read
		if time.Since(at) <= slowEvery {
			continue
		}
		if !found || at.Before(bestAt) {
			best, bestAt, found = e, at, true
		}
	}
	return best, found
}
