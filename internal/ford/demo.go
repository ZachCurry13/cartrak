package ford

import (
	"context"
	"encoding/json"
	"math"
	"sync"
	"time"
)

// Demo is a fake car for trying the app without a Ford account. Each call to
// Fetch advances a 24-step cycle: parked, driving, parked, charging.
type Demo struct {
	mu   sync.Mutex
	step int
	odo  float64
	soc  float64
}

func NewDemo() *Demo { return &Demo{odo: 12000, soc: 82} }

func (d *Demo) Name() string { return "demo" }
func (d *Demo) Ready() bool  { return true }

func (d *Demo) Fetch(ctx context.Context) (*State, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	phase := d.step % 24
	d.step++
	ign, charge, plug, kw, spd := "OFF", "NotReady", "Disconnected", 0.0, 0.0
	gear := "PARK"
	switch {
	case phase >= 4 && phase < 12: // driving
		ign, gear = "ON", "DRIVE"
		spd = 45 + 40*math.Abs(math.Sin(float64(phase)))
		d.odo += spd / 60 * 2
		d.soc -= 0.7
	case phase >= 16: // charging
		plug, charge, kw = "Connected", "ChargingAC", 10.5
		d.soc = math.Min(100, d.soc+3)
		if d.soc >= 100 {
			charge, kw = "ChargeCompleted", 0
		}
	}
	t := float64(d.step)
	lat, lon := 42.30+0.002*math.Sin(t/3), -83.20+0.003*math.Cos(t/3)
	doc := map[string]any{
		"info": map[string]any{
			"vehicleDetails": map[string]any{"odometer": d.odo, "batteryChargeLevel": map[string]any{"value": d.soc, "distanceToEmpty": d.soc * 3.8}},
			"vehicleStatus": map[string]any{
				"ignitionStatus": map[string]any{"value": ign}, "chargingStatus": map[string]any{"value": charge},
				"plugStatus": map[string]any{"value": plug == "Connected"}, "lockStatus": map[string]any{"value": "LOCKED"},
				"tirePressureWarning": false,
			},
		},
		"location": map[string]any{"vehicleLocation": map[string]any{"latitude": lat, "longitude": lon, "speed": spd}},
		// The real telemetry layout: vehicles[].metrics.<name>, with list metrics for wheels and doors.
		"telemetry": map[string]any{"vehicles": []any{map[string]any{"metrics": map[string]any{
			"tirePressure": []any{
				map[string]any{"vehicleWheel": "FRONT_LEFT", "value": 255.0}, map[string]any{"vehicleWheel": "FRONT_RIGHT", "value": 253.0},
				map[string]any{"vehicleWheel": "REAR_LEFT", "value": 260.0}, map[string]any{"vehicleWheel": "REAR_RIGHT", "value": 258.0}},
			"tirePressureStatus": []any{
				map[string]any{"vehicleWheel": "FRONT_LEFT", "value": "NORMAL"}, map[string]any{"vehicleWheel": "REAR_RIGHT", "value": "NORMAL"}},
			"doorLockStatus": []any{
				map[string]any{"vehicleDoor": "UNSPECIFIED_FRONT", "vehicleSide": "DRIVER", "value": lockWord(phase)},
				map[string]any{"vehicleDoor": "UNSPECIFIED_FRONT", "vehicleSide": "PASSENGER", "value": lockWord(phase)}},
			"doorStatus": []any{
				map[string]any{"vehicleDoor": "UNSPECIFIED_FRONT", "vehicleSide": "DRIVER", "value": "CLOSED"},
				map[string]any{"vehicleDoor": "TAILGATE", "value": tailgate(phase)}},
			"hoodStatus": map[string]any{"value": "CLOSED"},
			"indicators": map[string]any{"tirePressureWarning": map[string]any{"value": false}, "brakePadWearWarning": map[string]any{"value": phase == 8}},
		}}}},
		"XEV_BATTERY_TEMPERATURE":      map[string]any{"value": 24.5},
		"XEV_BATTERY_ENERGY_REMAINING": map[string]any{"value": d.soc * 0.88},
		"GEAR_LEVER_POSITION":          map[string]any{"value": gear},
		"AMBIENT_TEMP":                 map[string]any{"value": 18.0},
		"BATTERY_VOLTAGE":              map[string]any{"value": 12.6},
		"ACCELERATOR_PEDAL_POSITION":   map[string]any{"value": spd / 4},
	}
	raw, _ := json.Marshal(doc)
	s := ParseState(raw, time.Now())
	s.ChargeKw = ptr(kw)
	return s, nil
}

// lockWord and tailgate vary a little with the demo's cycle so the cards show different states.
func lockWord(phase int) string {
	if phase >= 4 && phase < 12 {
		return "UNLOCKED" // while "driving"
	}
	return "LOCKED"
}

func tailgate(phase int) string {
	if phase == 18 {
		return "AJAR"
	}
	return "CLOSED"
}
