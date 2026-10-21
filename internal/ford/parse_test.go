package ford

import (
	"testing"
	"time"
)

func TestParseRealTelemetryShape(t *testing.T) {
	raw := []byte(`{"telemetry":{"vehicles":[{"vin":"X","metrics":{
	  "xevBatteryStateOfCharge":{"value":72.5},
	  "xevBatteryRange":{"value":260.4},
	  "odometer":{"value":10234.5},
	  "ignitionStatus":{"value":"OFF"},
	  "xevPlugChargerStatus":{"value":"DISCONNECTED"},
	  "xevBatteryChargeDisplayStatus":{"value":"NOT_READY"},
	  "outsideTemperature":{"value":12.5},
	  "batteryVoltage":{"value":12.7},
	  "speed":{"value":10},
	  "doorLockStatus":{"value":"LOCKED"},
	  "position":{"value":{"location":{"lat":42.31,"lon":-83.2,"alt":190}}}
	}}]}}`)
	s := ParseState(raw, time.Now())
	if s.SoCPct == nil || *s.SoCPct != 72.5 || s.RangeKm == nil || *s.RangeKm != 260.4 || s.OdometerKm == nil || *s.OdometerKm != 10234.5 {
		t.Fatalf("battery/odometer not parsed: soc=%v range=%v odo=%v", s.SoCPct, s.RangeKm, s.OdometerKm)
	}
	if s.Lat == nil || *s.Lat != 42.31 || s.Lon == nil || *s.Lon != -83.2 {
		t.Fatalf("location not parsed: %v %v", s.Lat, s.Lon)
	}
	if s.Ignition != "OFF" || s.Plug != "DISCONNECTED" || s.ChargeState != "NOT_READY" {
		t.Fatalf("status not parsed: %+v", s)
	}
	if s.OutsideC == nil || *s.OutsideC != 12.5 || s.Battery12V == nil || *s.Battery12V != 12.7 || s.SpeedKph == nil || *s.SpeedKph != 36 {
		t.Fatalf("temperature/voltage/speed not parsed: %v %v %v", s.OutsideC, s.Battery12V, s.SpeedKph)
	}
	if s.Locked == nil || !*s.Locked {
		t.Fatalf("lock state not parsed: %v", s.Locked)
	}
}

// Tires and locks come as lists in the real telemetry.
func TestParseTiresAndLocksFromLists(t *testing.T) {
	raw := []byte(`{"telemetry":{"vehicles":[{"metrics":{
	  "tirePressure":[{"vehicleWheel":"FRONT_LEFT","value":255},{"vehicleWheel":"FRONT_RIGHT","value":251},
	                   {"vehicleWheel":"REAR_LEFT","value":262},{"vehicleWheel":"REAR_RIGHT","value":260}],
	  "doorLockStatus":[{"vehicleDoor":"UNSPECIFIED_FRONT","vehicleSide":"DRIVER","value":"LOCKED"},
	                    {"vehicleDoor":"UNSPECIFIED_FRONT","vehicleSide":"PASSENGER","value":"LOCKED"},
	                    {"vehicleDoor":"TAILGATE","value":"UNKNOWN"}]
	}}]}}`)
	s := ParseState(raw, time.Now())
	want := [4]float64{255, 251, 262, 260}
	for i, w := range want {
		if s.Tires[i] == nil || *s.Tires[i] != w {
			t.Fatalf("tire %d = %v, want %v", i, s.Tires[i], w)
		}
	}
	if s.Locked == nil || !*s.Locked {
		t.Fatalf("all known doors locked should be locked: %v", s.Locked)
	}

	one := []byte(`{"m":{"doorLockStatus":[{"vehicleDoor":"UNSPECIFIED_FRONT","value":"LOCKED"},{"vehicleDoor":"REAR_LEFT","value":"UNLOCKED"}]}}`)
	if s := ParseState(one, time.Now()); s.Locked == nil || *s.Locked {
		t.Fatalf("one unlocked door means not locked: %v", s.Locked)
	}
	all := []byte(`{"m":{"doorLockStatus":[{"vehicleDoor":"ALL_DOORS","value":"DOUBLE_LOCKED"}]}}`)
	if s := ParseState(all, time.Now()); s.Locked == nil || !*s.Locked {
		t.Fatalf("ALL_DOORS locked: %v", s.Locked)
	}
	none := ParseState([]byte(`{"m":{"speed":{"value":1}}}`), time.Now())
	if none.Locked != nil || none.Tires[0] != nil {
		t.Fatalf("absent metrics must stay unknown: %v %v", none.Locked, none.Tires[0])
	}
}

// What a fast charger looks like in real telemetry: the charge status says IN_PROGRESS, there is no onboard charger
// power, and only the battery's own current and voltage show that energy is flowing in.
func TestFastChargingIsRecognizedFromTheStatusWordAndThePackCurrent(t *testing.T) {
	raw := []byte(`{"telemetry":{"metrics":{
	  "xevBatteryChargeDisplayStatus":{"value":"IN_PROGRESS"},
	  "xevPlugChargerStatus":{"value":"CONNECTED"},
	  "xevBatteryIoCurrent":{"value":-110.2},
	  "xevBatteryVoltage":{"value":372.0},
	  "ignitionStatus":{"value":"ON"},
	  "speed":{"value":0}
	}}}`)
	s := ParseState(raw, time.Now())
	if !s.Charging() {
		t.Fatalf("IN_PROGRESS is charging: %+v", s)
	}
	if s.ChargeKw == nil || *s.ChargeKw < 40.9 || *s.ChargeKw > 41.1 {
		t.Fatalf("110.2 A at 372 V is about 41 kW: %v", s.ChargeKw)
	}
	for _, not := range []string{"NOT_READY", "PAUSED", "FAULT", "STATION_NOT_DETECTED", "COMPLETE", ""} {
		if ChargeStateCharging(not) {
			t.Errorf("%q is not charging", not)
		}
	}
	// Braking pushes current into the pack too, but with no plug it is not charging.
	driving := ParseState([]byte(`{"telemetry":{"metrics":{"xevPlugChargerStatus":{"value":"DISCONNECTED"},"xevBatteryIoCurrent":{"value":-60},"xevBatteryVoltage":{"value":370}}}}`), time.Now())
	if driving.ChargeKw != nil || driving.Charging() {
		t.Fatalf("regen is not charging: %+v", driving.ChargeKw)
	}
}
