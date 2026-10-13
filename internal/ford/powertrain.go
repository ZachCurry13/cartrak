package ford

import "strings"

// Powertrain is what kind of car this is. Ford's garage data says it (engineType, for example
// "BEV"). If that is not known yet it is worked out from the metrics the car reports (see Caps),
// and an admin can override both.
type Powertrain string

const (
	Unknown Powertrain = ""
	EV      Powertrain = "ev"   // battery electric
	PHEV    Powertrain = "phev" // plug-in hybrid
	HEV     Powertrain = "hev"  // regular hybrid, no plug
	Gas     Powertrain = "gas"  // petrol or diesel engine only
)

// CapsKV and EngineKV are the stored values that remember what the car has told us.
const (
	CapsKV   = "vehicle_caps"
	EngineKV = "vehicle_engine_type"
)

// Valid reports whether p is one of the known types.
func (p Powertrain) Valid() bool { return p == EV || p == PHEV || p == HEV || p == Gas }

// HasBattery is true when the car has a traction battery to show. Unknown shows everything.
func (p Powertrain) HasBattery() bool { return p != Gas }

// CanPlug is true when the car can be charged from the grid.
func (p Powertrain) CanPlug() bool { return p == EV || p == PHEV || p == Unknown }

// HasEngine is true when the car has a fuel engine (and so oil and fuel to show).
func (p Powertrain) HasEngine() bool { return p != EV }

// Label is the plain-language name.
func (p Powertrain) Label() string {
	switch p {
	case EV:
		return "Electric"
	case PHEV:
		return "Plug-in hybrid"
	case HEV:
		return "Hybrid"
	case Gas:
		return "Gas"
	}
	return "Not known yet"
}

// FromEngineType maps the garage "engineType" Ford reports (BEV, PHEV, HEV, ICE...) to a type.
func FromEngineType(s string) Powertrain {
	u := strings.ToUpper(strings.TrimSpace(s))
	has := func(parts ...string) bool {
		for _, p := range parts {
			if strings.Contains(u, p) {
				return true
			}
		}
		return false
	}
	switch {
	case u == "":
		return Unknown
	case has("PHEV", "PLUG"):
		return PHEV
	case u == "BEV" || u == "EV" || (has("ELECTRIC") && !has("HYBRID")):
		return EV
	case has("HEV", "HYBRID"):
		return HEV
	case u == "ICE" || has("GAS", "DIESEL", "PETROL", "FUEL"):
		return Gas
	}
	return Unknown
}

// EngineTypeOf reads the garage engine type from a flattened response ("" if not there yet).
func EngineTypeOf(f map[string]any) string { return strAt(f, "garage.engineType") }

// Caps are the kinds of data a car has been seen to report. They only ever grow, so one
// reading that happens to miss a metric cannot change the answer. Battery cars also list oil
// life, engine temperature and hybrid mode, so those are not evidence of an engine.
type Caps struct{ EV, Fuel, Plug bool }

// Or combines what has been seen so far with a new reading.
func (c Caps) Or(o Caps) Caps { return Caps{c.EV || o.EV, c.Fuel || o.Fuel, c.Plug || o.Plug} }

// Powertrain decides the type from what has been seen.
func (c Caps) Powertrain() Powertrain {
	switch {
	case c.EV && c.Fuel:
		if c.Plug {
			return PHEV
		}
		return HEV
	case c.EV:
		return EV
	case c.Fuel:
		return Gas
	}
	return Unknown
}

// String and ParseCaps keep Caps in one small settings value, for example "ev,plug".
func (c Caps) String() string {
	var p []string
	for _, x := range []struct {
		on   bool
		name string
	}{{c.EV, "ev"}, {c.Fuel, "fuel"}, {c.Plug, "plug"}} {
		if x.on {
			p = append(p, x.name)
		}
	}
	return strings.Join(p, ",")
}

func ParseCaps(s string) Caps {
	var c Caps
	for _, p := range strings.Split(s, ",") {
		switch strings.TrimSpace(p) {
		case "ev":
			c.EV = true
		case "fuel":
			c.Fuel = true
		case "plug":
			c.Plug = true
		}
	}
	return c
}

// CapsOf reads which kinds of data one reading contains. Fuel only counts when the tank level,
// range or volume is above zero (or fuel was used on a trip).
func CapsOf(f map[string]any) Caps {
	return Caps{
		EV: present(f, "xevBatteryStateOfCharge", "xevBatteryRange", "xevBatteryVoltage", "xevBatteryCapacity",
			"xevBatteryEnergyRemaining", "xevBatteryPerformanceStatus", "xevTractionMotorVoltage"),
		Fuel: positive(f, "fuelLevel", "fuelRange", "fuelVolume", "tripFuelConsumed"),
		Plug: present(f, "xevPlugChargerStatus", "xevBatteryChargeDisplayStatus", "xevChargeStationCommunicationStatus"),
	}
}

// present is true when any of the metrics has a real value (not empty, unknown or unsupported).
func present(f map[string]any, names ...string) bool {
	for _, n := range names {
		if v, ok := lookup(f, n); ok && real(v) {
			return true
		}
	}
	return false
}

func positive(f map[string]any, names ...string) bool {
	for _, n := range names {
		if v, ok := lookup(f, n); ok {
			if x, isNum := num(v); isNum && x > 0 {
				return true
			}
		}
	}
	return false
}

func real(v any) bool {
	s, isStr := v.(string)
	if !isStr {
		return v != nil
	}
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "", "UNKNOWN", "UNSUPPORTED", "INVALID", "UNAVAILABLE", "NOT_AVAILABLE", "NULL":
		return false
	}
	return true
}
