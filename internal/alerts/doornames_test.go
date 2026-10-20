package alerts

import (
	"strings"
	"testing"
)

func TestAnUnsplitFrontDoorIsNotCalledTheDriversDoor(t *testing.T) {
	p := "telemetry.metrics.doorStatus"
	f := map[string]any{
		p + "[0].value": "AJAR", p + "[0].vehicleDoor": "UNSPECIFIED_FRONT", p + "[0].vehicleSide": "DRIVER",
		p + "[1].value": "AJAR", p + "[1].vehicleDoor": "REAR_RIGHT",
	}
	got := strings.Join(openThings(f), ", ")
	if !strings.Contains(got, "front door (driver or passenger side)") || !strings.Contains(got, "rear right door") || strings.Contains(got, "driver door") {
		t.Fatalf("got %q", got)
	}
}
