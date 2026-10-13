package ford

import (
	"context"
	"encoding/json"
	"math"
	"strings"
	"sync"
	"time"
)

// Demo is a fake car for trying the app without a Ford account. Each call to Fetch advances a
// 24-step cycle: parked, driving, parked, charging (or refuelling). It uses the same layout the
// real API returns (telemetry.metrics.<name>) and can pretend to be an electric, gas, hybrid or
// plug-in hybrid car, so every part of the app can be checked for each type.
type Demo struct {
	mu   sync.Mutex
	kind Powertrain
	step int
	odo  float64
	soc  float64
	fuel float64
}

// NewDemo is the default demo: a battery-electric car.
func NewDemo() *Demo { return NewDemoKind("") }

// NewDemoKind makes a demo of the given type ("ev", "gas", "hev" or "phev"; anything else is "ev").
func NewDemoKind(kind string) *Demo {
	k := Powertrain(strings.ToLower(strings.TrimSpace(kind)))
	if !k.Valid() {
		k = EV
	}
	return &Demo{kind: k, odo: 12000, soc: 82, fuel: 70}
}

func (d *Demo) Name() string { return "demo" }
func (d *Demo) Ready() bool  { return true }

func (d *Demo) Fetch(ctx context.Context) (*State, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	phase := d.step % 24
	d.step++
	m := demoMetrics{kind: d.kind, phase: phase, odo: d.odo, soc: d.soc, fuel: d.fuel, ign: "OFF", plug: "DISCONNECTED", charge: "NOT_READY", gear: "PARK"}
	switch {
	case phase >= 4 && phase < 12: // driving
		m.ign, m.gear = "ON", "DRIVE"
		m.speed = (45 + 40*math.Abs(math.Sin(float64(phase)))) / 3.6 // m/s, as Ford reports it
		d.odo += m.speed * 3.6 / 60 * 2
		d.soc -= 0.7
		d.fuel -= 0.8
	case phase >= 16: // charging, or filling up
		if d.kind.CanPlug() {
			m.plug, m.charge, m.amps = "CONNECTED", "CHARGING", 43
			d.soc = math.Min(100, d.soc+3)
			if d.soc >= 100 {
				m.charge, m.amps = "CHARGE_COMPLETED", 0
			}
		}
		if d.kind.HasEngine() && d.kind != EV && phase == 16 {
			d.fuel = 90
		}
	}
	m.odo, m.soc, m.fuel = d.odo, d.soc, d.fuel
	t := float64(d.step)
	m.lat, m.lon = 42.30+0.002*math.Sin(t/3), -83.20+0.003*math.Cos(t/3)
	raw, _ := json.Marshal(map[string]any{
		"telemetry": map[string]any{"metrics": m.metrics()},
		"garage":    map[string]any{"engineType": m.engineType(), "make": "Demo", "modelName": "Demo car", "modelYear": "2024"},
	})
	return ParseState(raw, time.Now()), nil
}
