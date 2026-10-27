package server

import (
	"net/http"
	"sync"
	"time"
)

// The savings page works out, for every charge, what the same range would have cost in the gas car. That reads every trip and
// looks up a price per charge, so the answer is kept for a short while and thrown away whenever a charge or a setting is saved.
const savingsKeep = 45 * time.Second

type savingsCache struct {
	mu   sync.Mutex
	at   time.Time
	body map[string]any
}

func (c *savingsCache) get() (map[string]any, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.body == nil || time.Since(c.at) > savingsKeep {
		return nil, false
	}
	return c.body, true
}

func (c *savingsCache) put(b map[string]any) {
	c.mu.Lock()
	c.at, c.body = time.Now(), b
	c.mu.Unlock()
}

func (c *savingsCache) drop() {
	c.mu.Lock()
	c.body = nil
	c.mu.Unlock()
}

// forgetSavings wraps a handler that changes what the savings figures depend on (a charge, a setting).
func (s *Server) forgetSavings(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h(w, r)
		s.savingsKept.drop()
	}
}

// stillUsingOwnPrice says whether any charge's gas cost fell back to the owner's own price while live prices are on. That is what
// a page shows until the place and the weekly price have been fetched in the background, so such an answer is not kept.
func (s *Server) stillUsingOwnPrice(eq map[int64]chargeGas) bool {
	if !s.liveGasOn() {
		return false
	}
	for _, g := range eq {
		if g.Source == ownGasSource {
			return true
		}
	}
	return false
}
