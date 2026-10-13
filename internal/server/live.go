package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/ZachCurry13/cartrak/internal/poller"
	"github.com/ZachCurry13/cartrak/internal/store"
)

// summary is the small version of a reading: enough for the home tiles and
// the live view, without the long list of raw fields. Used by data saver mode.
type summary struct {
	TS          int64    `json:"ts"`
	SoCPct      *float64 `json:"socPct"`
	RangeKm     *float64 `json:"rangeKm"`
	SpeedKph    *float64 `json:"speedKph"`
	Ignition    string   `json:"ignition"`
	ChargeState string   `json:"chargeState"`
	Plug        string   `json:"plug"`
	ChargeKw    *float64 `json:"chargeKw"`
	OutsideC    *float64 `json:"outsideC"`
	OdometerKm  *float64 `json:"odometerKm"`
	Lat         *float64 `json:"lat"`
	Lon         *float64 `json:"lon"`
	Locked      *bool    `json:"locked"`
	FuelPct     *float64 `json:"fuelPct"`
	OilLifePct  *float64 `json:"oilLifePct"`
	EnergyKwh   *float64 `json:"energyKwh"`   // energy left in the traction battery
	CapacityKwh *float64 `json:"capacityKwh"` // its estimated capacity
}

func summarize(x *store.Sample) *summary {
	if x == nil {
		return nil
	}
	return &summary{TS: x.TS, SoCPct: x.SoCPct, RangeKm: x.RangeKm, SpeedKph: x.SpeedKph, Ignition: x.Ignition,
		ChargeState: x.ChargeState, Plug: x.Plug, ChargeKw: x.ChargeKw, OutsideC: x.OutsideC,
		OdometerKm: x.OdometerKm, Lat: x.Lat, Lon: x.Lon, Locked: x.Locked, FuelPct: x.FuelPct, OilLifePct: x.OilLifePct, EnergyKwh: x.EnergyKwh, CapacityKwh: x.CapacityKwh}
}

// liveEvent is pushed to every open live connection after each new reading.
type liveEvent struct {
	Sample *summary      `json:"sample"`
	Trip   *store.Trip   `json:"trip"`
	Status poller.Status `json:"status"`
}

// hub fans events out to connected browsers. A slow client just misses
// events (its buffer is small and sends never block the poller).
type hub struct {
	mu   sync.Mutex
	subs map[chan liveEvent]struct{}
}

func (h *hub) subscribe() (chan liveEvent, func()) {
	ch := make(chan liveEvent, 4)
	h.mu.Lock()
	if h.subs == nil {
		h.subs = map[chan liveEvent]struct{}{}
	}
	h.subs[ch] = struct{}{}
	h.mu.Unlock()
	return ch, func() {
		h.mu.Lock()
		delete(h.subs, ch)
		h.mu.Unlock()
	}
}

func (h *hub) publish(e liveEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs {
		select {
		case ch <- e:
		default:
		}
	}
}

func (h *hub) count() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.subs)
}

func (s *Server) currentEvent() liveEvent {
	latest, _ := s.S.LatestSample()
	trip, _ := s.S.OpenTrip()
	return liveEvent{Sample: summarize(latest), Trip: trip, Status: s.P.Status()}
}

// Broadcast pushes the newest reading to everyone watching live.
// Wire it to the poller's After hook.
func (s *Server) Broadcast() {
	if s.hub.count() == 0 {
		return
	}
	s.hub.publish(s.currentEvent())
}

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
