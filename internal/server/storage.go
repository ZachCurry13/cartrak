package server

import (
	"context"
	"net/http"
	"sync"
	"time"
)

type trimState struct {
	mu      sync.Mutex
	Running bool   `json:"running"`
	Done    int    `json:"done"`
	Total   int    `json:"total"`
	Message string `json:"message"`
}

func (t *trimState) snapshot() trimState {
	t.mu.Lock()
	defer t.mu.Unlock()
	return trimState{Running: t.Running, Done: t.Done, Total: t.Total, Message: t.Message}
}

// storageInfo is the database size, how many readings are older than the chosen age, and any trim in progress.
// GET /api/storage?days=90. Admin only.
func (s *Server) storageInfo(w http.ResponseWriter, r *http.Request) {
	days := intQuery(r, "days", 90, 30, 3650)
	snaps, samples, err := s.S.StorageCounts(time.Now().AddDate(0, 0, -days).Unix())
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, map[string]any{"dbBytes": s.S.DBSize(), "days": days, "snapshots": snaps, "samples": samples, "trim": s.trim.snapshot()})
}

// storageTrim starts removing the raw Ford copy of readings older than `days` (at least 30), in the background.
// POST /api/storage/trim {"days":90,"confirm":"TRIM"}. Admin only.
func (s *Server) storageTrim(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Days    int    `json:"days"`
		Confirm string `json:"confirm"`
	}
	if decode(r, &in) != nil || in.Confirm != "TRIM" || in.Days < 30 || in.Days > 3650 {
		fail(w, 400, "choose at least 30 days and type TRIM to confirm")
		return
	}
	cutoff := time.Now().AddDate(0, 0, -in.Days).Unix()
	snaps, samples, err := s.S.StorageCounts(cutoff)
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	s.trim.mu.Lock()
	if s.trim.Running {
		s.trim.mu.Unlock()
		fail(w, http.StatusConflict, "a trim is already running")
		return
	}
	s.trim.Running, s.trim.Done, s.trim.Total, s.trim.Message = true, 0, snaps+samples, "Working…"
	s.trim.mu.Unlock()
	go func() {
		err := s.S.TrimRaw(context.Background(), cutoff, 300, 150*time.Millisecond, func(n int) {
			s.trim.mu.Lock()
			s.trim.Done = n
			s.trim.mu.Unlock()
		})
		s.trim.mu.Lock()
		s.trim.Running = false
		if err != nil {
			s.trim.Message = "Stopped: " + err.Error()
		} else {
			s.trim.Message = "Done. The space is reused for new readings."
		}
		s.trim.mu.Unlock()
	}()
	ok(w, "started")
}
