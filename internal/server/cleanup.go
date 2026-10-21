package server

import (
	"context"
	"log"
	"strconv"
	"time"

	"github.com/ZachCurry13/cartrak/internal/settings"
)

// CleanupLoop removes phone GPS points older than the "keep days" setting, once at start and then
// every six hours. With the setting empty (the default) nothing is ever removed.
func (s *Server) CleanupLoop(ctx context.Context) {
	go s.noticeLoop(ctx)  // badges and the monthly Wrapped, hourly
	go s.serviceLoop(ctx) // service visits, every few minutes
	for {
		s.cleanupOnce(time.Now())
		select {
		case <-ctx.Done():
			return
		case <-time.After(6 * time.Hour):
		}
	}
}

func (s *Server) cleanupOnce(now time.Time) {
	s.purgeTiles(now)
	if s.Cfg == nil {
		return
	}
	days, err := strconv.Atoi(s.Cfg.Value(settings.KeepDaysKey))
	if err != nil || days <= 0 {
		return
	}
	if n, err := s.S.DeletePointsBefore(now.AddDate(0, 0, -days)); err != nil {
		log.Printf("phone point cleanup failed: %v", err)
	} else if n > 0 {
		log.Printf("removed %d phone GPS points older than %d days", n, days)
	}
}
