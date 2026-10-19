package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
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
	// The battery and drive motor, as the car reports them. Each is nil when this car does not send it.
	BatteryTempC  *float64 `json:"batteryTempC"`
	PackVolts     *float64 `json:"packVolts"`
	Battery12V    *float64 `json:"battery12v"`
	MotorVolts    *float64 `json:"motorVolts"`
	MotorAmps     *float64 `json:"motorAmps"`
	MotorTempC    *float64 `json:"motorTempC"`    // only if the car reports a motor or inverter temperature
	PackAmps      *float64 `json:"packAmps"`      // current into (negative) or out of (positive) the battery
	CoolantC      *float64 `json:"coolantC"`      // the coolant temperature the car reports
	TimeToFullMin *float64 `json:"timeToFullMin"` // the car's own estimate while charging
	ChargerKwh    *float64 `json:"chargerKwh"`    // energy the car measured going in this session
}

func summarize(x *store.Sample) *summary {
	if x == nil {
		return nil
	}
	extra := map[string]any{}
	_ = json.Unmarshal(x.Extra, &extra)
	return &summary{BatteryTempC: ford.Number(extra, "xevBatteryTemperature"), PackVolts: x.HVVolts, Battery12V: x.Battery12V,
		MotorVolts: ford.Number(extra, "xevTractionMotorVoltage"), MotorAmps: ford.Number(extra, "xevTractionMotorCurrent"), MotorTempC: motorTemp(extra), PackAmps: ford.Number(extra, "xevBatteryIoCurrent"), CoolantC: ford.Number(extra, "engineCoolantTemp"), TimeToFullMin: ford.Number(extra, "xevBatteryTimeToFullCharge"), ChargerKwh: ford.Number(extra, "xevBatteryChargerEnergyOutput"), TS: x.TS, SoCPct: x.SoCPct, RangeKm: x.RangeKm, SpeedKph: x.SpeedKph, Ignition: x.Ignition,
		ChargeState: x.ChargeState, Plug: x.Plug, ChargeKw: x.ChargeKw, OutsideC: x.OutsideC,
		OdometerKm: x.OdometerKm, Lat: x.Lat, Lon: x.Lon, Locked: x.Locked, FuelPct: x.FuelPct, OilLifePct: x.OilLifePct, EnergyKwh: x.EnergyKwh, CapacityKwh: x.CapacityKwh}
}

// liveEvent is pushed to every open live connection after each new reading.
type liveEvent struct {
	Sample *summary      `json:"sample"`
	Trip   *store.Trip   `json:"trip"`
	Status poller.Status `json:"status"`
	// DrivingToday is how long the car has been driven since local midnight, in seconds, including a trip still going.
	DrivingToday int64 `json:"drivingToday"`
	// CarTS is when the car itself last reported something new (0 if unknown), unlike Sample.TS, which is when CarTrak last asked.
	CarTS int64 `json:"carTs"`
	// Charge is the charging session in progress, if any.
	Charge *store.Charge `json:"charge"`
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
	openCharge, _ := s.S.OpenCharge()
	return liveEvent{Sample: summarize(latest), Trip: trip, Status: s.P.Status(), DrivingToday: s.drivingToday(), CarTS: s.S.CarLastReport(), Charge: openCharge}
}

// drivingToday adds up the time spent on trips since midnight in the car's time zone, leaving out trips that went nowhere.
func (s *Server) drivingToday() int64 {
	now := time.Now()
	y, m, d := now.In(s.defaultLoc()).Date()
	midnight := time.Date(y, m, d, 0, 0, 0, 0, s.defaultLoc()).Unix()
	ts, err := s.S.SpanTrips(midnight)
	if err != nil {
		return 0
	}
	var total int64
	for _, t := range ts {
		if !t.Open && t.DistanceKm < 0.2 && t.MaxSpeedKph < 8 {
			continue // a trip that went nowhere (sitting in the car) is not driving
		}
		from, to := t.StartTS, now.Unix()
		if t.EndTS != nil {
			to = *t.EndTS
		}
		if from < midnight {
			from = midnight
		}
		if to > from {
			total += to - from
		}
	}
	return total
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
