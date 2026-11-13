package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ZachCurry13/cartrak/internal/nws"
	"github.com/ZachCurry13/cartrak/internal/push"
)

// wxCache keeps the warnings for each rounded area for ten minutes.
type wxCache struct {
	mu   sync.Mutex
	byAt map[string]wxEntry
}

type wxEntry struct {
	at   time.Time
	list []nws.Alert
	err  error
}

// alertsAt is the warnings in force at a place, asked at most every ten minutes for its area.
func (s *Server) alertsAt(ctx context.Context, lat, lon float64) ([]nws.Alert, error) {
	key := fmt.Sprintf("%.1f,%.1f", nws.Round(lat), nws.Round(lon))
	c := &s.wx
	c.mu.Lock()
	if c.byAt == nil {
		c.byAt = map[string]wxEntry{}
	}
	if e, ok := c.byAt[key]; ok && time.Since(e.at) < 10*time.Minute {
		c.mu.Unlock()
		return e.list, e.err
	}
	c.mu.Unlock()
	list, err := nws.Active(ctx, nil, s.NWSURL, lat, lon)
	c.mu.Lock()
	if len(c.byAt) > 200 {
		c.byAt = map[string]wxEntry{}
	}
	c.byAt[key] = wxEntry{at: time.Now(), list: list, err: err}
	c.mu.Unlock()
	return list, err
}

// weatherAlerts: GET /api/weather-alerts?points=44.9,-93.2;45.1,-93.4 (up to 12 places; the car's position when left out).
func (s *Server) weatherAlerts(w http.ResponseWriter, r *http.Request) {
	if s.Cfg == nil || !s.Cfg.WeatherAlerts() {
		ok(w, map[string]any{"on": false, "places": []any{}})
		return
	}
	type place struct {
		Lat    float64     `json:"lat"`
		Lon    float64     `json:"lon"`
		Alerts []nws.Alert `json:"alerts"`
		Error  string      `json:"error,omitempty"`
	}
	var pts [][2]float64
	for _, p := range strings.Split(r.URL.Query().Get("points"), ";") {
		a, b, found := strings.Cut(p, ",")
		lat, e1 := strconv.ParseFloat(strings.TrimSpace(a), 64)
		lon, e2 := strconv.ParseFloat(strings.TrimSpace(b), 64)
		if found && e1 == nil && e2 == nil && lat >= -90 && lat <= 90 && lon >= -180 && lon <= 180 {
			pts = append(pts, [2]float64{lat, lon})
		}
	}
	if len(pts) == 0 {
		if x, _ := s.S.LatestSample(); x != nil && x.Lat != nil && x.Lon != nil {
			pts = append(pts, [2]float64{*x.Lat, *x.Lon})
		}
	}
	if len(pts) > 12 {
		pts = pts[:12]
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	out := []place{}
	for _, p := range pts {
		list, err := s.alertsAt(ctx, p[0], p[1])
		pl := place{Lat: p[0], Lon: p[1], Alerts: list}
		if pl.Alerts == nil {
			pl.Alerts = []nws.Alert{}
		}
		if err != nil {
			pl.Error = err.Error()
		}
		out = append(out, pl)
	}
	ok(w, map[string]any{"on": true, "places": out})
}

const kvWxSent = "wx_sent" // the warnings already sent, with when: a warning is told about once

// weatherWatch sends the road warnings where the car is, and the evening heads-up about tomorrow's appointments.
func (s *Server) weatherWatch(ctx context.Context, now time.Time) {
	if s.Cfg == nil {
		return
	}
	s.dayAheadOnce(ctx, now)
	if !s.Cfg.WeatherAlerts() {
		return
	}
	x, _ := s.S.LatestSample()
	if x == nil || x.Lat == nil || x.Lon == nil {
		return
	}
	list, err := s.alertsAt(ctx, *x.Lat, *x.Lon)
	if err != nil {
		return
	}
	sent := map[string]int64{}
	_ = json.Unmarshal([]byte(s.S.GetKV(kvWxSent)), &sent)
	for id, at := range sent {
		if now.Unix()-at > 7*86400 {
			delete(sent, id)
		}
	}
	for _, a := range list {
		if !nws.ForDriving(a) || sent[a.ID] != 0 || (a.Ends != 0 && a.Ends < now.Unix()) {
			continue
		}
		sent[a.ID] = now.Unix()
		body := a.Headline
		if body == "" {
			body = a.Event + " for " + a.Area
		}
		s.tell(ctx, push.Alert{Kind: push.WeatherWarning, Title: a.Event, Body: body, URL: "#/"})
	}
	b, _ := json.Marshal(sent)
	_ = s.S.SetKV(kvWxSent, string(b))
}
