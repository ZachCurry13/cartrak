package ford

import "math"

// demoMetrics is one step of the demo car, turned into the metrics a real car reports.
type demoMetrics struct {
	kind                    Powertrain
	phase                   int
	odo, soc, fuel, speed   float64
	amps                    float64
	lat, lon                float64
	ign, plug, charge, gear string
}

func v(x any) map[string]any { return map[string]any{"value": x} }

func (m demoMetrics) engineType() string {
	return map[Powertrain]string{EV: "BEV", PHEV: "PHEV", HEV: "HEV", Gas: "ICE"}[m.kind]
}

func (m demoMetrics) metrics() map[string]any {
	locked := "LOCKED"
	if m.phase >= 4 && m.phase < 12 {
		locked = "UNLOCKED"
	}
	tailgate := "CLOSED"
	if m.phase == 18 {
		tailgate = "AJAR"
	}
	out := map[string]any{
		"ignitionStatus": v(m.ign), "gearLeverPosition": v(m.gear), "speed": v(m.speed), "odometer": v(m.odo * 1.0),
		"outsideTemperature": v(18.0), "ambientTemp": v(0), "batteryVoltage": v(12.1 + 0.4*math.Abs(math.Sin(float64(m.phase)/5))),
		"batteryStateOfCharge": v(78), "batteryLoadStatus": v("NORMAL_OPERATION"),
		"position":   map[string]any{"value": map[string]any{"location": map[string]any{"lat": m.lat, "lon": m.lon}}},
		"hoodStatus": v("CLOSED"), "engineSpeed": v(0),
		"doorLockStatus": []any{
			map[string]any{"vehicleDoor": "UNSPECIFIED_FRONT", "vehicleSide": "DRIVER", "value": locked},
			map[string]any{"vehicleDoor": "UNSPECIFIED_FRONT", "vehicleSide": "PASSENGER", "value": locked}},
		"doorStatus": []any{
			map[string]any{"vehicleDoor": "UNSPECIFIED_FRONT", "vehicleSide": "DRIVER", "value": "CLOSED"},
			map[string]any{"vehicleDoor": "TAILGATE", "value": tailgate}},
		"tirePressure": []any{
			map[string]any{"vehicleWheel": "FRONT_LEFT", "value": 255.0, "wheelPlacardFront": 241.0},
			map[string]any{"vehicleWheel": "FRONT_RIGHT", "value": 253.0, "wheelPlacardFront": 241.0},
			map[string]any{"vehicleWheel": "REAR_LEFT", "value": 260.0, "wheelPlacardRear": 241.0},
			map[string]any{"vehicleWheel": "REAR_RIGHT", "value": 258.0, "wheelPlacardRear": 241.0}},
		"tirePressureStatus": []any{
			map[string]any{"vehicleWheel": "FRONT_LEFT", "value": "NORMAL"}, map[string]any{"vehicleWheel": "REAR_RIGHT", "value": "NORMAL"}},
		"indicators": map[string]any{"tirePressureWarning": v(false), "brakePadWearWarning": v(m.phase == 8)},
	}
	if m.kind != Gas { // a traction battery
		out["xevBatteryStateOfCharge"] = v(m.soc)
		out["xevBatteryActualStateOfCharge"] = v(m.soc - 4.5)
		out["xevBatteryRange"] = v(m.soc * 3.8)
		out["xevBatteryEnergyRemaining"] = v(m.soc * 0.88)
		out["xevBatteryCapacity"] = v(88.4)
		out["xevBatteryVoltage"] = v(371.0)
		out["xevBatteryTemperature"] = v(24.5)
		out["xevBatteryPerformanceStatus"] = v("NORMAL")
		out["xevBatteryIoCurrent"] = v(0.08)
		out["xevTractionMotorVoltage"] = v(8)
	}
	if m.kind.CanPlug() && m.kind != Unknown {
		out["xevPlugChargerStatus"] = v(m.plug)
		out["xevBatteryChargeDisplayStatus"] = v(m.charge)
		out["xevBatteryChargerVoltageOutput"] = v(240)
		out["xevBatteryChargerCurrentOutput"] = v(m.amps)
	}
	switch m.kind {
	case EV:
		// A real battery car lists these too, with meaningless values; the app must ignore them.
		out["oilLifeRemaining"] = v(100)
		out["hybridVehicleModeStatus"] = v("AUTO_CHARGE_DEPLETE")
		out["engineCoolantTemp"] = v(15)
	default:
		out["oilLifeRemaining"] = v(62)
		out["fuelLevel"] = v(m.fuel)
		out["fuelRange"] = v(m.fuel * 6.2)
		out["engineCoolantTemp"] = v(88)
		if m.kind != Gas {
			out["hybridVehicleModeStatus"] = v("AUTO_CHARGE_DEPLETE")
		}
	}
	return out
}
