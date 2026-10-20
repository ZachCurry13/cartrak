package server

import (
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

// localStart is the trip's start time on the clock where it happened.
func localStart(t *store.Trip) string {
	loc := time.UTC
	if t.TZ != "" {
		if l, err := time.LoadLocation(t.TZ); err == nil {
			loc = l
		}
	}
	return time.Unix(t.StartTS, 0).In(loc).Format("2006-01-02 15:04:05")
}
