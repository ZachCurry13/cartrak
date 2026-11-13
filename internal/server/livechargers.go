package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/ZachCurry13/cartrak/internal/googleplaces"
)

// liveCache keeps each answer about free plugs for five minutes: they change slowly, and each question to Google is billed.
var liveCache = struct {
	sync.Mutex
	m map[string]liveEntry
}{m: map[string]liveEntry{}}

type liveEntry struct {
	at   time.Time
	list []googleplaces.Charger
}

const liveKeep = 5 * time.Minute

// liveChargers: GET /api/chargers/live?lat=45&lon=-93&r=2000. The chargers within r meters (500 to 50000, 2000 when left out),
// nearest first, with how many plugs their networks report free. Asked only when someone taps, never in the background.
func (s *Server) liveChargers(w http.ResponseWriter, r *http.Request) {
	if s.Cfg == nil || s.Cfg.GooglePlacesKey() == "" {
		fail(w, 400, "Live charger availability needs the Google Places key (More, Integrations, Google Places)")
		return
	}
	q := r.URL.Query()
	lat, e1 := strconv.ParseFloat(q.Get("lat"), 64)
	lon, e2 := strconv.ParseFloat(q.Get("lon"), 64)
	if e1 != nil || e2 != nil || lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		fail(w, 400, "give a position")
		return
	}
	radius := 2000.0
	if v, err := strconv.ParseFloat(q.Get("r"), 64); err == nil {
		radius = min(max(v, 500), 50000)
	}
	key := fmt.Sprintf("%.3f,%.3f,%.0f", lat, lon, radius)
	liveCache.Lock()
	e, hit := liveCache.m[key]
	liveCache.Unlock()
	if hit && time.Since(e.at) < liveKeep {
		ok(w, map[string]any{"chargers": e.list, "asked": e.at.Unix()})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	list, err := googleplaces.Chargers(ctx, nil, s.PlacesURL, s.Cfg.GooglePlacesKey(), lat, lon, radius)
	if err != nil {
		fail(w, http.StatusBadGateway, err.Error())
		return
	}
	now := time.Now()
	liveCache.Lock()
	if len(liveCache.m) > 500 {
		liveCache.m = map[string]liveEntry{}
	}
	liveCache.m[key] = liveEntry{at: now, list: list}
	liveCache.Unlock()
	ok(w, map[string]any{"chargers": list, "asked": now.Unix()})
}
