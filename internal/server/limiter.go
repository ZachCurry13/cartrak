package server

import (
	"sync"
	"time"
)

// limiter blocks an IP for a minute after 5 failed logins.
type limiter struct {
	mu    sync.Mutex
	fails map[string][]time.Time
}

func (l *limiter) blocked(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	cut := time.Now().Add(-time.Minute)
	var keep []time.Time
	for _, t := range l.fails[ip] {
		if t.After(cut) {
			keep = append(keep, t)
		}
	}
	if l.fails == nil {
		l.fails = map[string][]time.Time{}
	}
	if len(keep) == 0 {
		delete(l.fails, ip)
	} else {
		l.fails[ip] = keep
	}
	if len(l.fails) > 1000 { // many addresses (a scan of share links): drop every one that has gone quiet
		for k, ts := range l.fails {
			if len(ts) == 0 || !ts[len(ts)-1].After(cut) {
				delete(l.fails, k)
			}
		}
	}
	return len(keep) >= 5
}

func (l *limiter) fail(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.fails == nil {
		l.fails = map[string][]time.Time{}
	}
	l.fails[ip] = append(l.fails[ip], time.Now())
}
