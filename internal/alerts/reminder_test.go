package alerts

import (
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
)

func TestThePlugInReminderComesOnceInTheEveningWhenItIsNeeded(t *testing.T) {
	e, r, _ := setup(t)
	now := t0.Add(8 * time.Hour) // 20:00
	e.Now = func() time.Time { return now }
	e.ReminderTime = func() string { return "21:00" }
	e.ReminderBelow = func() float64 { return 60 }
	e.Loc = func() *time.Location { return time.UTC }
	low := func(s *ford.State) { v := 45.0; s.SoCPct = &v }
	at := func(min int, mod func(*ford.State)) { e.Handle(reading(min, mod)); e.Tick() }

	at(480, low) // 20:00, before the reminder time
	if len(r.got) != 0 {
		t.Fatalf("too early: %s", r.kinds())
	}
	now = t0.Add(9*time.Hour + 5*time.Minute)
	at(545, low)
	if r.kinds() != "plug_reminder" || r.got[0].URL != "#/charging" {
		t.Fatalf("a low parked car that is not plugged in: %s", r.kinds())
	}
	now = now.Add(10 * time.Minute)
	at(555, low)
	if len(r.got) != 1 {
		t.Fatalf("only one a day: %s", r.kinds())
	}
}

func TestThePlugInReminderStaysQuietWhenThereIsNothingToSay(t *testing.T) {
	cases := map[string]func(*ford.State){
		"plenty of charge": func(s *ford.State) { v := 85.0; s.SoCPct = &v },
		"already plugged":  func(s *ford.State) { v := 45.0; s.SoCPct = &v; s.Plug, s.ChargeState = "CONNECTED", "CHARGING" },
		"plug unknown":     func(s *ford.State) { v := 45.0; s.SoCPct = &v; s.Plug = "" },
		"no battery level": func(s *ford.State) { s.SoCPct = nil },
	}
	for name, mod := range cases {
		e, r, _ := setup(t)
		now := t0.Add(9*time.Hour + 5*time.Minute)
		e.Now = func() time.Time { return now }
		e.ReminderTime = func() string { return "21:00" }
		e.ReminderBelow = func() float64 { return 60 }
		e.Handle(reading(545, mod))
		e.Tick()
		if len(r.got) != 0 {
			t.Errorf("%s: %s", name, r.kinds())
		}
	}
	// Off, outside the window, or a reading that is hours old.
	e, r, _ := setup(t)
	now := t0.Add(9*time.Hour + 5*time.Minute)
	e.Now = func() time.Time { return now }
	e.ReminderBelow = func() float64 { return 60 }
	e.ReminderTime = func() string { return "" }
	e.Handle(reading(545, func(s *ford.State) { v := 45.0; s.SoCPct = &v }))
	e.Tick()
	e.ReminderTime = func() string { return "21:00" }
	now = t0.Add(12 * time.Hour) // 00:00, well after the window
	e.Tick()
	now = t0.Add(9*time.Hour + 30*time.Minute)
	e.prev.TS = t0 // a stale reading
	e.Tick()
	if len(r.got) != 0 {
		t.Fatalf("off, late and stale: %s", r.kinds())
	}
}
