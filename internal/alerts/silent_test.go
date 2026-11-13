package alerts

import (
	"strings"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
)

// stamped makes the car's newest report this old at the time of the reading.
func stamped(ago time.Duration) func(*ford.State) {
	return func(s *ford.State) {
		s.Extra["telemetry.metrics.speed.updateTime"] = s.TS.Add(-ago).UTC().Format(time.RFC3339)
	}
}

func TestACarThatHasSaidNothingForADayIsReportedOncePerDay(t *testing.T) {
	e, r, _ := setup(t)
	e.Handle(reading(0, stamped(3*time.Hour))) // asleep for a few hours: normal
	if len(r.got) != 0 {
		t.Fatalf("a few hours of quiet is a sleeping car: %v", r.kinds())
	}
	e.Handle(reading(10, stamped(30*time.Hour)))
	e.Handle(reading(20, stamped(30*time.Hour)))
	if r.kinds() != "car_silent" || !strings.Contains(r.got[0].Body, " ago") || !strings.Contains(r.got[0].Body, "12 V") {
		t.Fatalf("one alert that says how long and why it may be: %v %v", r.kinds(), r.got)
	}
}

func TestTheLow12VAlertSaysHowOldItsReadingIs(t *testing.T) {
	e, r, _ := setup(t)
	e.Handle(reading(0, func(s *ford.State) {
		v := 11.6
		s.Battery12V = &v
		s.Extra["telemetry.metrics.batteryVoltage.updateTime"] = e.now().Add(-20 * time.Hour).UTC().Format(time.RFC3339)
	}))
	if len(r.got) != 1 || r.got[0].Kind != "low_12v" || !strings.Contains(r.got[0].Body, "last reported this 20 hours ago") {
		t.Fatalf("%v", r.got)
	}
}

func TestAZeroPercentBesideAHealthyVoltageIsNotALow12VBattery(t *testing.T) {
	e, r, _ := setup(t)
	e.Handle(reading(0, func(s *ford.State) {
		v := 12.2
		s.Battery12V = &v
		s.Extra["telemetry.metrics.batteryStateOfCharge.value"] = 0.0
	}))
	if len(r.got) != 0 {
		t.Fatalf("12.2 V with a 0%% that the car never fills in is fine: %v", r.got)
	}
	e2, r2, _ := setup(t)
	e2.Handle(reading(0, func(s *ford.State) {
		v := 12.2
		s.Battery12V = &v
		s.Extra["telemetry.metrics.batteryStateOfCharge.value"] = 30.0
	}))
	if r2.kinds() != "low_12v" {
		t.Fatalf("a real low percent still alerts: %v", r2.got)
	}
}
