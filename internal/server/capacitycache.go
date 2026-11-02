package server

import (
	"sync"
	"time"
)

const (
	capacityWindowDays = 21 // the capacity the car estimates wanders a few percent with the weather and charge level, so use the middle of three weeks
	capacityMinReads   = 5  // fewer readings than this and the latest one is all there is
	capacityTTL        = 10 * time.Minute
)

// capacityCache holds the smoothed battery capacity so the many calls that need it do not each read three weeks of readings.
type capacityCache struct {
	mu   sync.Mutex
	at   time.Time
	kwh  float64
	have bool
}

// steadyCapacity is the middle capacity reading of the last weeks, or false when there are too few readings.
func (s *Server) steadyCapacity() (float64, bool) {
	c := &s.capCache
	c.mu.Lock()
	defer c.mu.Unlock()
	if time.Since(c.at) < capacityTTL {
		return c.kwh, c.have
	}
	m, n, err := s.S.RecentMedian("capacityKwh", time.Now().AddDate(0, 0, -capacityWindowDays).Unix())
	c.at, c.have, c.kwh = time.Now(), err == nil && n >= capacityMinReads && m > 0, m
	return c.kwh, c.have
}
