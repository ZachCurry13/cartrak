package poller

import (
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
)

func TestAFastChargerWithNoReportedPowerAndAPausedChargeAreReadAtTheChargingPace(t *testing.T) {
	p := &Poller{Iv: Intervals{Driving: 30 * time.Second, Charging: 40 * time.Second, Parked: 10 * time.Minute}}
	dc := &ford.State{Ignition: "OFF", Plug: "CONNECTED", ChargeState: "IN_PROGRESS"} // no kilowatts at all
	if got := p.interval(dc); got != 40*time.Second {
		t.Fatalf("charging by state: %v", got)
	}
	paused := &ford.State{Ignition: "OFF", Plug: "CONNECTED", ChargeState: "NOT_READY"}
	if got := p.interval(paused); got != 40*time.Second {
		t.Fatalf("a pause just after charging keeps the pace: %v", got)
	}
	unplugged := &ford.State{Ignition: "OFF", Plug: "DISCONNECTED", ChargeState: "NOT_READY"}
	if got := p.interval(unplugged); got != 10*time.Minute {
		t.Fatalf("unplugged is parked: %v", got)
	}
	p.chargeSeen = time.Now().Add(-30 * time.Minute)
	if got := p.interval(paused); got != 10*time.Minute {
		t.Fatalf("a long pause falls back to the parked pace: %v", got)
	}
}
