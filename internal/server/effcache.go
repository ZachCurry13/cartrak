package server

import (
	"sync"
	"time"

	"github.com/ZachCurry13/cartrak/internal/savings"
)

// The distance a kWh has been giving reads every trip, and the savings page, the day view and the charging list all ask for it, so
// it is kept for a minute (a new trip then shows up within the minute).
const effKeep = time.Minute

type effCache struct {
	mu  sync.Mutex
	at  time.Time
	val *savings.Efficiency
}

func (c *effCache) get() *savings.Efficiency {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.val != nil && time.Since(c.at) < effKeep {
		return c.val
	}
	return nil
}

func (c *effCache) put(e *savings.Efficiency) {
	c.mu.Lock()
	c.at, c.val = time.Now(), e
	c.mu.Unlock()
}
