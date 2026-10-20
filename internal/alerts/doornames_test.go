package alerts

import (
	"github.com/ZachCurry13/cartrak/internal/ford"
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

func TestLeftOpenSaysWhenTheCarLastReportedAndDoesNotRepeatAnOldReport(t *testing.T) {
	e, r, _ := setup(t)
	p := "telemetry.metrics.doorStatus"
	stale := func(s *ford.State) {
		s.Extra[p+"[0].value"] = "AJAR"
		s.Extra[p+"[0].vehicleDoor"] = "REAR_RIGHT"
		s.Extra[p+"[0].updateTime"] = "2026-10-20T14:29:55Z"
	}
	for i := 0; i < 6; i++ {
		e.Handle(reading(i*4, stale))
	}
	if len(r.got) != 1 || !strings.Contains(r.got[0].Body, "last reported this at") || !strings.Contains(r.got[0].Body, "may have been closed since") {
		t.Fatalf("one alert that says how old the report is: %s %v", r.kinds(), r.got)
	}
	for i := 0; i < 8; i++ { // a drive in between, then the very same old report again
		e.Handle(reading(40+i*4, func(s *ford.State) { stale(s); s.Ignition = "ON" }))
	}
	for i := 0; i < 6; i++ {
		e.Handle(reading(80+i*4, stale))
	}
	if len(r.got) != 1 {
		t.Fatalf("the same old report must not alert twice: %s", r.kinds())
	}
}
