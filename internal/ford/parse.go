package ford

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Flatten turns any JSON document into dotted-path -> scalar pairs, e.g.
// "info.vehicleStatus.ignitionStatus.value" -> "ON". Arrays use [i].
func Flatten(raw []byte) map[string]any {
	var v any
	out := map[string]any{}
	if json.Unmarshal(raw, &v) != nil {
		return out
	}
	flat("", v, out)
	addMetricAliases(out)
	return out
}

func flat(p string, v any, out map[string]any) {
	switch t := v.(type) {
	case map[string]any:
		for k, c := range t {
			flat(join(p, k), c, out)
		}
	case []any:
		for i, c := range t {
			flat(fmt.Sprintf("%s[%d]", p, i), c, out)
		}
	default:
		if t != nil {
			out[p] = t
		}
	}
}

func join(p, k string) string {
	if p == "" {
		return k
	}
	return p + "." + k
}

// lookup finds the first flattened value whose path ends with one of the
// suffixes (case-insensitive, matched on whole dotted segments).
// norm lowercases and drops underscores, so the dictionary name XEV_BATTERY_RANGE and the
// telemetry key xevBatteryRange compare equal.
func norm(s string) string { return strings.ToLower(strings.ReplaceAll(s, "_", "")) }

func lookup(f map[string]any, suffixes ...string) (any, bool) {
	for _, s := range suffixes {
		ls := norm(s)
		for p, v := range f {
			lp := norm(p)
			if lp == ls || strings.HasSuffix(lp, "."+ls) ||
				lp == ls+".value" || strings.HasSuffix(lp, "."+ls+".value") {
				return v, true
			}
		}
	}
	return nil, false
}

func num(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case string:
		x, err := strconv.ParseFloat(t, 64)
		return x, err == nil
	}
	return 0, false
}

func numAt(f map[string]any, suffixes ...string) *float64 {
	if v, ok := lookup(f, suffixes...); ok {
		if x, ok := num(v); ok {
			return &x
		}
	}
	return nil
}

func strAt(f map[string]any, suffixes ...string) string {
	if v, ok := lookup(f, suffixes...); ok {
		switch t := v.(type) {
		case string:
			return t
		case bool:
			return strconv.FormatBool(t)
		case float64:
			return strconv.FormatFloat(t, 'f', -1, 64)
		}
	}
	return ""
}

// ParseState maps a raw provider response onto State. It accepts FordConnect
// v1 field names and the data-dictionary metric names. Speeds in the
// dictionary are m/s and are converted to km/h; FordConnect v1 speed is
// already km/h.
func ParseState(raw []byte, at time.Time) *State {
	s := ParseFlat(Flatten(raw), at)
	s.Raw = raw
	return s
}

// ParseFlat is ParseState for a reading that is already flattened (as stored with each sample).
func ParseFlat(f map[string]any, at time.Time) *State {
	s := &State{TS: at, Extra: f}
	s.OdometerKm = numAt(f, "vehicleDetails.odometer", "odometer")
	s.SoCPct = numAt(f, "batteryChargeLevel.value", "XEV_BATTERY_STATE_OF_CHARGE", "batteryFillLevel.value")
	s.RangeKm = numAt(f, "batteryChargeLevel.distanceToEmpty", "XEV_BATTERY_RANGE", "elVehDTE.value")
	s.Lat = numAt(f, "vehicleLocation.latitude", "position.value.location.lat", "position.location.lat", "position.latitude", "latitude")
	s.Lon = numAt(f, "vehicleLocation.longitude", "position.value.location.lon", "position.location.lon", "position.longitude", "longitude")
	s.SpeedKph = numAt(f, "vehicleLocation.speed")
	if s.SpeedKph == nil {
		if ms := numAt(f, "SPEED.value", "SPEED"); ms != nil {
			s.SpeedKph = ptr(*ms * 3.6)
		}
	}
	s.Ignition = strAt(f, "ignitionStatus.value", "IGNITION_STATUS")
	s.ChargeState = strAt(f, "chargingStatus.value", "XEV_BATTERY_CHARGE_DISPLAY_STATUS")
	s.Plug = plug(f)
	s.OutsideC = numAt(f, "outsideTemperature", "AMBIENT_TEMP") // ambientTemp reads 0 on some cars; outsideTemperature is the real one
	s.Battery12V = numAt(f, "BATTERY_VOLTAGE", "battery.batteryStatusActual.value")
	if v := numAt(f, "XEV_BATTERY_CHARGER_VOLTAGE_OUTPUT"); v != nil {
		if a := numAt(f, "XEV_BATTERY_CHARGER_CURRENT_OUTPUT"); a != nil {
			s.ChargeKw = ptr(*v * *a / 1000)
		}
	}
	// Fast (DC) charging reports no onboard-charger power. The battery's own current and voltage always do: a negative
	// current is charge going in. Only trusted while the plug is connected, since braking also pushes current in.
	if (s.ChargeKw == nil || *s.ChargeKw < 0.2) && plugConnected(s.Plug) {
		if i := numAt(f, "xevBatteryIoCurrent"); i != nil && *i <= -1 {
			if v := numAt(f, "xevBatteryVoltage"); v != nil {
				s.ChargeKw = ptr(-*i * *v / 1000)
			}
		}
	}
	if st := strAt(f, "lockStatus.value", "DOOR_LOCK_STATUS"); st != "" {
		s.Locked = ptr(strings.EqualFold(st, "LOCKED") || strings.EqualFold(st, "DOUBLE_LOCKED"))
	}
	parseTires(f, s) // list metrics in the real telemetry layout
	parseLocks(f, s)
	parseHealth(f, s)
	return s
}

func plug(f map[string]any) string {
	v, ok := lookup(f, "plugStatus.value", "XEV_PLUG_CHARGER_STATUS")
	if !ok {
		return ""
	}
	switch t := v.(type) {
	case bool:
		if t {
			return "Connected"
		}
		return "Disconnected"
	case string:
		return t
	}
	return ""
}
