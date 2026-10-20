package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// liveStream is a server-sent-events feed: one message per new reading,
// plus a heartbeat so proxies (Cloudflare) keep the connection open.
func (s *Server) liveStream(w http.ResponseWriter, r *http.Request) {
	fl, ok := w.(http.Flusher)
	if !ok {
		fail(w, 500, "streaming unsupported")
		return
	}
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("X-Accel-Buffering", "no")
	send := func(e liveEvent) error {
		b, _ := json.Marshal(e)
		_, err := fmt.Fprintf(w, "event: reading\ndata: %s\n\n", b)
		fl.Flush()
		return err
	}
	ch, cancel := s.hub.subscribe()
	defer cancel()
	if send(s.currentEvent()) != nil {
		return
	}
	beat := time.NewTicker(25 * time.Second)
	defer beat.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case e := <-ch:
			if send(e) != nil {
				return
			}
		case <-beat.C:
			if _, err := fmt.Fprint(w, ": ping\n\n"); err != nil {
				return
			}
			fl.Flush()
		}
	}
}

// setLiveMode switches road-trip mode (faster polling while driving).
func (s *Server) setLiveMode(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Enabled bool `json:"enabled"`
	}
	if decode(r, &b) != nil {
		fail(w, 400, "bad request")
		return
	}
	s.P.SetLive(b.Enabled)
	ok(w, s.P.Status())
}

// motorTemp finds a drive motor or inverter temperature among the car's fields, if it reports one. Ford's
// documented list has none for most cars, so this is nil unless the car sends something with motor or inverter and
// temperature in its name.
func motorTemp(extra map[string]any) *float64 {
	best := ""
	for k := range extra {
		lk := strings.ToLower(k)
		if strings.HasSuffix(lk, "updatetime") || !strings.Contains(lk, "temp") || !(strings.Contains(lk, "motor") || strings.Contains(lk, "inverter")) {
			continue
		}
		if best == "" || k < best {
			best = k
		}
	}
	if best == "" {
		return nil
	}
	switch v := extra[best].(type) {
	case float64:
		return &v
	}
	return nil
}
