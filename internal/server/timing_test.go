package server

import (
	"strings"
	"testing"
	"time"
)

func TestTheSlowestRequestsAreListedWithoutIdsInThePath(t *testing.T) {
	timingMu.Lock()
	timings = map[string]*reqStat{}
	timingMu.Unlock()
	noteTiming("GET /api/trips/{id}", 40*time.Millisecond)
	noteTiming("GET /api/trips/{id}", 60*time.Millisecond)
	noteTiming("GET /api/latest", 5*time.Millisecond)
	got := SlowestRequests()
	if len(got) != 2 || !strings.HasPrefix(got[0], "GET /api/trips/{id}") || !strings.Contains(got[0], "average    50 ms, slowest    60 ms, 2 times") {
		t.Fatalf("slowest first, with the average and the worst: %q", got)
	}
}
