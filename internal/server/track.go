package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

// Tracking: the owner switches it on, uses the app, switches it off and exports everything that happened in that time. While it is
// on, every request the app makes and every page, tap and error the phone reports is recorded; the readings, trips, charges,
// decisions and alerts are already kept, and the export picks the ones in the window. A forgotten session stops by itself.

const (
	kvTrack       = "track_session" // "from|until|stoppedAt", stoppedAt 0 while it runs
	trackDefault  = 4               // hours before it stops by itself
	trackMaxHours = 12
	maxTrackBatch = 300
)

type trackSession struct {
	mu                  sync.Mutex
	loaded              bool
	from, until, stopAt int64
}

// trackWindow is the tracking session as stored: read once, then kept.
func (s *Server) trackWindow() (from, until, stopAt int64) {
	t := &s.tracking
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.loaded {
		t.loaded = true
		f := strings.Split(s.S.GetKV(kvTrack), "|")
		if len(f) == 3 {
			t.from, _ = strconv.ParseInt(f[0], 10, 64)
			t.until, _ = strconv.ParseInt(f[1], 10, 64)
			t.stopAt, _ = strconv.ParseInt(f[2], 10, 64)
		}
	}
	return t.from, t.until, t.stopAt
}

func (s *Server) setTrack(from, until, stopAt int64) {
	s.trackWindow()
	t := &s.tracking
	t.mu.Lock()
	t.from, t.until, t.stopAt = from, until, stopAt
	t.mu.Unlock()
	_ = s.S.SetKV(kvTrack, fmt.Sprintf("%d|%d|%d", from, until, stopAt))
}

// trackOn says whether a session is running now.
func (s *Server) trackOn() bool {
	_, until, stopAt := s.trackWindow()
	return stopAt == 0 && until > time.Now().Unix()
}

// trackSpan is the stretch the last session covers: from its start to when it stopped, timed out or now.
func (s *Server) trackSpan() (from, to int64, have bool) {
	from, until, stopAt := s.trackWindow()
	if from == 0 {
		return 0, 0, false
	}
	to = min(until, time.Now().Unix())
	if stopAt > 0 {
		to = stopAt
	}
	return from, to, true
}

func (s *Server) trackView(w http.ResponseWriter, r *http.Request) {
	from, to, have := s.trackSpan()
	_, until, _ := s.trackWindow()
	n := 0
	if have {
		es, _ := s.S.TrackEventsBetween(from, to)
		n = len(es)
	}
	ok(w, map[string]any{"active": s.trackOn(), "from": from, "to": to, "until": until, "events": n, "has": have})
}

func (s *Server) trackStart(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Hours int `json:"hours"`
	}
	_ = decode(r, &in)
	if in.Hours < 1 || in.Hours > trackMaxHours {
		in.Hours = trackDefault
	}
	now := time.Now().Unix()
	s.setTrack(now, now+int64(in.Hours)*3600, 0)
	s.S.LogDecision("tracking", fmt.Sprintf("tracking started for up to %d hours", in.Hours))
	s.trackView(w, r)
}

func (s *Server) trackStop(w http.ResponseWriter, r *http.Request) {
	from, until, stopAt := s.trackWindow()
	if from > 0 && stopAt == 0 {
		s.setTrack(from, until, min(until, time.Now().Unix()))
		s.S.LogDecision("tracking", "tracking stopped")
	}
	s.trackView(w, r)
}

func (s *Server) trackExtend(w http.ResponseWriter, r *http.Request) {
	from, until, stopAt := s.trackWindow()
	if s.trackOn() {
		s.setTrack(from, min(until+3600, time.Now().Unix()+trackMaxHours*3600), stopAt)
	}
	s.trackView(w, r)
}

// trackEvents: POST /api/track/events. The phone's page views, taps and errors, kept only while a session runs.
func (s *Server) trackEvents(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Events []store.TrackEvent `json:"events"`
	}
	if decode(r, &in) != nil || !s.trackOn() {
		ok(w, map[string]any{"stored": 0})
		return
	}
	var keep []store.TrackEvent
	for _, e := range in.Events {
		if len(keep) >= maxTrackBatch || (e.Kind != "page" && e.Kind != "tap" && e.Kind != "error") {
			continue
		}
		e.A, e.B = clip(e.A, 200), clip(e.B, 200)
		if e.TS <= 0 || e.TS > time.Now().Unix()+60 {
			e.TS = time.Now().Unix()
		}
		keep = append(keep, e)
	}
	_ = s.S.AddTrackEvents(keep)
	ok(w, map[string]any{"stored": len(keep)})
}

func clip(v string, n int) string {
	if len(v) > n {
		return v[:n]
	}
	return v
}

type statusWriter struct {
	http.ResponseWriter
	code int
}

func (w *statusWriter) WriteHeader(c int) { w.code = c; w.ResponseWriter.WriteHeader(c) }

// tracked records every request the app makes while a session runs: the method, path and query, the answer's status and how long it
// took. Nothing in a request's body is kept (it could hold a password or a key).
func (s *Server) tracked(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if !strings.HasPrefix(p, "/api/") || strings.HasPrefix(p, "/api/track") || strings.HasPrefix(p, "/api/live") || strings.HasPrefix(p, "/api/tiles") || !s.trackOn() {
			h.ServeHTTP(w, r)
			return
		}
		sw := &statusWriter{ResponseWriter: w, code: 200}
		start := time.Now()
		h.ServeHTTP(sw, r)
		a := r.Method + " " + p
		if r.URL.RawQuery != "" {
			a += "?" + r.URL.RawQuery
		}
		_ = s.S.AddTrackEvents([]store.TrackEvent{{TS: start.Unix(), Kind: "request", A: clip(a, 200), B: strconv.Itoa(sw.code), N: time.Since(start).Milliseconds()}})
	})
}
