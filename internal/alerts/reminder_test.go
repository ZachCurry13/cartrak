package alerts

import (
	"strings"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/store"
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

func TestAChargeThatStopsLowerThanUsualIsFlaggedButAnOwnersLimitIsNot(t *testing.T) {
	stop := func(e *Engine, soc float64) {
		e.Handle(reading(0, func(s *ford.State) { s.Plug, s.ChargeState = "CONNECTED", "CHARGING" }))
		e.Handle(reading(30, func(s *ford.State) { v := soc; s.SoCPct = &v; s.Plug, s.ChargeState = "CONNECTED", "COMPLETE" }))
	}
	e, r, _ := setup(t)
	stop(e, 45)
	if r.kinds() != "charge_interrupted" || !strings.Contains(r.got[0].Body, "45%") {
		t.Fatalf("no earlier sessions to compare with: %s", r.kinds())
	}

	e2, r2, s2 := setup(t)
	for i := 0; i < 3; i++ { // the owner always charges to about 50
		c := &store.Charge{StartTS: int64(1000 + i)}
		if err := s2.CreateCharge(c); err != nil {
			t.Fatal(err)
		}
		end, soc := int64(2000+i), 50.0+float64(i)
		c.EndTS, c.EndSoC, c.Open = &end, &soc, false
		s2.UpdateCharge(c)
	}
	stop(e2, 51)
	if r2.kinds() != "charge_complete" {
		t.Fatalf("stopping where it usually stops is complete: %s", r2.kinds())
	}
	e3, r3, _ := setup(t)
	stop(e3, 85)
	if r3.kinds() != "charge_complete" {
		t.Fatalf("a high stop is complete: %s", r3.kinds())
	}
}

func TestAServiceReminderAlertsOnceWhenDueAndAgainAfterItIsDone(t *testing.T) {
	e, r, s := setup(t)
	every, last := 10000.0, 1000.0
	m := &store.Maintenance{Name: "Tire rotation", EveryKm: &every, LastOdoKm: &last, LastTS: t0.Unix()}
	if err := s.SaveMaintenance(m); err != nil {
		t.Fatal(err)
	}
	at := func(min int, odo float64) { e.Handle(reading(min, func(st *ford.State) { st.OdometerKm = &odo })) }
	at(0, 5000)
	at(5, 10500) // not due yet: due at 11000
	if len(r.got) != 0 {
		t.Fatalf("too early: %s", r.kinds())
	}
	at(10, 11050)
	at(15, 11100)
	if r.kinds() != "maintenance_due" || !strings.Contains(r.got[0].Title, "Tire rotation") {
		t.Fatalf("once, when due: %s", r.kinds())
	}
	od := 11100.0
	s.MarkMaintenanceDone(m.ID, &od)
	at(20, 15000)
	at(25, 21200) // due again at 21100
	if r.kinds() != "maintenance_due,maintenance_due" {
		t.Fatalf("done re-arms it: %s", r.kinds())
	}
}

func TestArrivingAndLeavingANamedPlaceAreAnnouncedOnceWithoutEdgeWobble(t *testing.T) {
	e, r, s := setup(t)
	home := &store.NamedPlace{Name: "Home", Lat: 40, Lon: -75, RadiusM: 150}
	if err := s.SaveNamedPlace(home); err != nil {
		t.Fatal(err)
	}
	at := func(min int, lat float64) {
		e.Handle(reading(min, func(st *ford.State) { la, lo := lat, -75.0; st.Lat, st.Lon = &la, &lo }))
	}
	at(0, 40.0001) // the first reading after a start is the baseline: already home, nothing to say
	if len(r.got) != 0 {
		t.Fatalf("baseline: %s", r.kinds())
	}
	at(5, 40.0020)  // 220 m: beyond 150 m but inside the leaving margin (187 m)? no, 222 m is outside it
	at(10, 40.0016) // 178 m: near the edge but still out
	at(15, 40.0015) // 167 m
	if r.kinds() != "leave_place" || r.got[0].Title != "Left Home" {
		t.Fatalf("left once: %s", r.kinds())
	}
	at(20, 40.0005) // back inside
	at(25, 40.0016) // 178 m: outside the place but inside the margin, so still home
	if r.kinds() != "leave_place,arrive_place" {
		t.Fatalf("arrived, and the wobble at the edge says nothing: %s", r.kinds())
	}
}
