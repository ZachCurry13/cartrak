package server

import (
	"sync"
	"time"

	"github.com/ZachCurry13/cartrak/internal/charging"
)

// Home asks a dozen things at once, and half of them (the savings, the day, the charge summary, the records) each read and price every
// charging session. The database answers one question at a time, so those repeats queue behind each other and the page waits for all of
// them. The priced sessions are kept for a few seconds and shared, and thrown away when a charge, a place or a setting is saved.
const sessionsKeep = 20 * time.Second

type sessionsEntry struct {
	at   time.Time
	ss   []charging.Session
	info effInfo
}

type sessionsCache struct {
	mu sync.Mutex
	m  map[int]sessionsEntry
}

func (c *sessionsCache) get(limit int) ([]charging.Session, effInfo, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.m[limit]
	if !ok || time.Since(e.at) > sessionsKeep {
		return nil, effInfo{}, false
	}
	return append([]charging.Session(nil), e.ss...), e.info, true // a copy: a caller may reorder it
}

func (c *sessionsCache) put(limit int, ss []charging.Session, info effInfo) {
	c.mu.Lock()
	if c.m == nil {
		c.m = map[int]sessionsEntry{}
	}
	c.m[limit] = sessionsEntry{at: time.Now(), ss: append([]charging.Session(nil), ss...), info: info}
	c.mu.Unlock()
}

func (c *sessionsCache) drop() {
	c.mu.Lock()
	c.m = nil
	c.mu.Unlock()
}
