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

func TestTheWeeklySummaryComesOnceOnSundayEveningWithTheWeeksNumbers(t *testing.T) {
	e, r, s := setup(t)
	sunday := time.Date(2026, 10, 18, 19, 0, 0, 0, time.UTC) // a Sunday
	now := sunday.Add(-time.Hour)                            // 18:00? no: 18:00 is the hour, this is 18:00
	e.Now = func() time.Time { return now }
	e.Loc = func() *time.Location { return time.UTC }
	mk := func(daysAgo int, km, from, to float64) {
		start := sunday.AddDate(0, 0, -daysAgo)
		tr := &store.Trip{StartTS: start.Unix(), StartKwh: &from}
		if err := s.CreateTrip(tr); err != nil {
			t.Fatal(err)
		}
		end := start.Add(time.Hour).Unix()
		tr.EndTS, tr.EndKwh, tr.DistanceKm, tr.Open = &end, &to, km, false
		s.UpdateTrip(tr)
	}
	mk(1, 100, 80, 60) // 100 km on 20 kWh
	mk(3, 60, 60, 50)  // 60 km on 10 kWh
	mk(20, 500, 80, 10)
	e.Tick()
	if r.kinds() != "weekly_summary" || !strings.Contains(r.got[0].Body, "160 km") || !strings.Contains(r.got[0].Body, "2 trips") {
		t.Fatalf("%s %+v", r.kinds(), r.got)
	}
	now = now.Add(30 * time.Minute)
	e.Tick()
	now = sunday.AddDate(0, 0, -1)
	e.Tick() // a Saturday
	if len(r.got) != 1 {
		t.Fatalf("once a week, and only on Sunday: %s", r.kinds())
	}
}

func TestAJoltOrAnAlarmOnAParkedCarIsAlertedAndLogged(t *testing.T) {
	e, r, s := setup(t)
	e.BumpLevel = func() float64 { return 1.0 }
	at := func(min int, x, y, z float64, alarm string, mod func(*ford.State)) {
		e.Handle(reading(min, func(st *ford.State) {
			st.Extra["telemetry.metrics.acceleration.value.x"] = x
			st.Extra["telemetry.metrics.acceleration.value.y"] = y
			st.Extra["telemetry.metrics.acceleration.value.z"] = z
			if alarm != "" {
				st.Extra["telemetry.metrics.alarmStatus.value"] = alarm
			}
			if mod != nil {
				mod(st)
			}
		}))
	}
	at(0, 0.1, 0.1, 0, "ARMED", nil)     // baseline
	at(2, -0.19, -0.08, 0, "ARMED", nil) // ordinary parked noise
	if len(r.got) != 0 {
		t.Fatalf("noise is not a bump: %s", r.kinds())
	}
	at(4, 0.7, 0.4, 0.1, "ARMED", nil) // 0.8: logged as a candidate but under the level
	at(6, 1.5, 0.9, 0.3, "ARMED", nil) // 1.8: a bump
	at(8, 2.0, 1.0, 0.0, "ARMED", nil) // more shaking right after: one alert only
	if r.kinds() != "bump_parked" {
		t.Fatalf("one bump alert: %s", r.kinds())
	}
	at(30, 0.1, 0, 0, "TRIGGERED", nil)
	if r.kinds() != "bump_parked,alarm_sounding" {
		t.Fatalf("an alarm: %s", r.kinds())
	}
	ev, _ := s.RecentParkedEvents(10)
	if len(ev) != 4 { // 0.8 candidate, 1.8, 2.2 (logged, not alerted again), alarm
		t.Fatalf("the log keeps what was seen: %+v", ev)
	}
	st := s.ParkedWatchStats()
	if st.Checked < 6 || st.MaxSeen < 2.2 {
		t.Fatalf("totals: %+v", st)
	}
	// Driving jolts are normal and ignored, and level 0 switches it off.
	n := len(r.got)
	at(50, 5, 5, 5, "", func(s *ford.State) { sp := 60.0; s.SpeedKph = &sp; s.Ignition = "ON" })
	e.BumpLevel = func() float64 { return 0 }
	at(70, 5, 5, 5, "", nil)
	if len(r.got) != n {
		t.Fatalf("nothing while driving or switched off: %s", r.kinds())
	}
}

func TestALowDiskIsAlertedOnceEveryThreeDaysAndOnlyWhenLow(t *testing.T) {
	e, r, _ := setup(t)
	now := t0
	e.Now = func() time.Time { return now }
	const g = 1 << 30
	free, total := uint64(50*g), uint64(100*g)
	e.DiskSpace = func() (uint64, uint64, bool) { return free, total, true }
	e.Tick()
	if len(r.got) != 0 {
		t.Fatalf("plenty of room: %s", r.kinds())
	}
	free = 3 * g
	e.Tick()
	e.Tick()
	if r.kinds() != "low_disk" {
		t.Fatalf("low: once, not on every check: %s", r.kinds())
	}
	now = now.Add(80 * time.Hour)
	e.Tick()
	if r.kinds() != "low_disk,low_disk" {
		t.Fatalf("again after three days: %s", r.kinds())
	}
	e.DiskSpace = func() (uint64, uint64, bool) { return 0, 0, false } // not known on this system
	now = now.Add(100 * time.Hour)
	e.Tick()
	if len(r.got) != 2 {
		t.Fatalf("unknown is quiet: %s", r.kinds())
	}
}
