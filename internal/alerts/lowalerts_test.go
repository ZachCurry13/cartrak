package alerts

import (
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
)

func TestLowBatteryWarnsOnceThenRearmsAfterRecovery(t *testing.T) {
	e, r, _ := setup(t)
	e.LowBatteryLevel = func() float64 { return 20 }
	at := func(min int, soc float64, mod func(*ford.State)) {
		e.Handle(reading(min, func(s *ford.State) {
			s.SoCPct = &soc
			if mod != nil {
				mod(s)
			}
		}))
	}
	at(0, 60, nil) // baseline
	at(10, 25, nil)
	if len(r.got) != 0 {
		t.Fatalf("above the level: %s", r.kinds())
	}
	at(20, 19, nil)
	at(30, 18, nil)
	if r.kinds() != "low_battery" {
		t.Fatalf("once: %s", r.kinds())
	}
	at(40, 22, nil) // up a little, not five points above: still armed off
	at(50, 19, nil)
	if len(r.got) != 1 {
		t.Fatalf("no repeat until it has recovered: %s", r.kinds())
	}
	at(60, 30, nil)
	at(70, 19, nil)
	if len(r.got) != 2 {
		t.Fatalf("after recovering it warns again: %s", r.kinds())
	}
}

func TestLowBatteryIsQuietWhileChargingOffOrAfterARestart(t *testing.T) {
	e, r, _ := setup(t)
	e.LowBatteryLevel = func() float64 { return 20 }
	low := func(s *ford.State) { v := 12.0; s.SoCPct = &v }
	e.Handle(reading(0, low)) // the first reading after a restart only sets the baseline
	e.Handle(reading(10, low))
	if len(r.got) != 0 {
		t.Fatalf("restart while already low: %s", r.kinds())
	}
	e2, r2, _ := setup(t)
	e2.LowBatteryLevel = func() float64 { return 20 }
	e2.Handle(reading(0, func(s *ford.State) { v := 60.0; s.SoCPct = &v }))
	e2.Handle(reading(10, func(s *ford.State) { v := 12.0; s.SoCPct = &v; s.Plug, s.ChargeState = "CONNECTED", "CHARGING" }))
	if r2.kinds() != "plug_in" { // plugging in is its own alert; no low battery on top
		t.Fatalf("charging: %s", r2.kinds())
	}
	e3, r3, _ := setup(t)
	e3.Handle(reading(0, low))
	e3.Handle(reading(10, low))
	if len(r3.got) != 0 {
		t.Fatalf("off: %s", r3.kinds())
	}
}

func TestChargingBudgetWarnsOncePerMonth(t *testing.T) {
	e, r, _ := setup(t)
	now := t0
	e.Now = func() time.Time { return now }
	e.Loc = func() *time.Location { return time.UTC }
	spent := 40.0
	e.ChargeBudget = func() float64 { return 50 }
	e.MonthCost = func() (float64, bool) { return spent, true }
	e.Tick()
	if len(r.got) != 0 {
		t.Fatalf("under budget: %s", r.kinds())
	}
	spent = 55
	e.Tick()
	e.Tick()
	if r.kinds() != "charge_budget" {
		t.Fatalf("once: %s", r.kinds())
	}
	now = now.AddDate(0, 1, 0)
	e.Tick()
	if len(r.got) != 2 {
		t.Fatalf("next month it can warn again: %s", r.kinds())
	}
}
