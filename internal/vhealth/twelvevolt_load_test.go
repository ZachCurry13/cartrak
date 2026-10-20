package vhealth

import "testing"

func TestAccessoryModeIsNotAWarningOnHealth(t *testing.T) {
	for v, want := range map[string]string{"NORMAL_OPERATION": "ok", "ACCESSORY_MODE": "ok", "LOAD_SHED": "warn"} {
		s := twelveVolt(map[string]any{"telemetry.metrics.batteryLoadStatus.value": v}, Trends{}, Options{})
		var got string
		for _, r := range s.Rows {
			if r.Label == "Load" {
				got = r.State
			}
		}
		if got != want {
			t.Fatalf("%s: got %q, want %q", v, got, want)
		}
	}
}
