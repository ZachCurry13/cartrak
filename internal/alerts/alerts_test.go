package alerts

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/push"
	"github.com/ZachCurry13/cartrak/internal/store"
)

type rec struct{ got []push.Alert }

func (r *rec) Notify(_ context.Context, a push.Alert) { r.got = append(r.got, a) }
func (r *rec) kinds() string {
	var k []string
	for _, a := range r.got {
		k = append(k, a.Kind)
	}
	return strings.Join(k, ",")
}

func setup(t *testing.T) (*Engine, *rec, *store.Store) {
	s, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	r := &rec{}
	return &Engine{S: s, N: r}, r, s
}

var t0 = time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)

func reading(min int, mod func(*ford.State)) *ford.State {
	soc := 70.0
	st := &ford.State{TS: t0.Add(time.Duration(min) * time.Minute), Plug: "DISCONNECTED", ChargeState: "NOT_READY", SoCPct: &soc, Ignition: "OFF", Extra: map[string]any{}}
	if mod != nil {
		mod(st)
	}
	return st
}

func TestPlugAndChargeAlertsFollowChangesNotState(t *testing.T) {
	e, r, _ := setup(t)
	e.Handle(reading(0, func(s *ford.State) { s.Plug = "CONNECTED" })) // already plugged in at start: nothing to say
	if len(r.got) != 0 {
		t.Fatalf("a restart must not announce the current state: %s", r.kinds())
	}
	e.Handle(reading(5, func(s *ford.State) { s.Plug, s.ChargeState = "CONNECTED", "CHARGING" }))
	e.Handle(reading(60, func(s *ford.State) { s.Plug, s.ChargeState = "CONNECTED", "COMPLETE" }))
	if r.kinds() != "charge_complete" || !strings.Contains(r.got[0].Body, "Battery at 70%") {
		t.Fatalf("charging stopping while plugged in is 'complete': %s %+v", r.kinds(), r.got)
	}
	e.Handle(reading(70, nil)) // cable removed
	e.Handle(reading(80, func(s *ford.State) { s.Plug = "CONNECTED" }))
	if r.kinds() != "charge_complete,plug_out,plug_in" {
		t.Fatalf("plug alerts: %s", r.kinds())
	}
}

func TestUnpluggingMidChargeIsNotChargingComplete(t *testing.T) {
	e, r, _ := setup(t)
	e.Handle(reading(0, func(s *ford.State) { s.Plug, s.ChargeState = "CONNECTED", "CHARGING" }))
	e.Handle(reading(10, nil))
	if r.kinds() != "plug_out" {
		t.Fatalf("got %s", r.kinds())
	}
}

func TestLowTireIsAnnouncedOncePerProblem(t *testing.T) {
	e, r, _ := setup(t)
	low := func(s *ford.State) {
		s.Extra = ford.Flatten([]byte(`{"m":{"tirePressureStatus":[{"value":"LOW","vehicleWheel":"FRONT_LEFT"},{"value":"NORMAL","vehicleWheel":"REAR_RIGHT"}]}}`))
	}
	e.Handle(reading(0, nil))
	e.Handle(reading(5, low))
	e.Handle(reading(10, low)) // still low: no repeat
	if r.kinds() != "low_tire" || !strings.Contains(r.got[0].Body, "Front left tire") {
		t.Fatalf("got %s %+v", r.kinds(), r.got)
	}
	e.Handle(reading(15, nil)) // fixed
	e.Handle(reading(20, low)) // a new problem
	if r.kinds() != "low_tire,low_tire" {
		t.Fatalf("got %s", r.kinds())
	}
	e2, r2, _ := setup(t)
	e2.Handle(reading(0, low)) // already low when CarTrak starts
	if len(r2.got) != 0 {
		t.Fatal("a tire that was already low at startup should not alert again after every restart")
	}
}

func TestTwelveVoltAlertIsOnceADayAndSurvivesARestart(t *testing.T) {
	e, r, s := setup(t)
	now := t0
	e.Now = func() time.Time { return now }
	volts := func(v float64) func(*ford.State) { return func(st *ford.State) { st.Battery12V = &v } }

	e.Handle(reading(0, volts(12.4)))
	e.Handle(reading(5, volts(11.6)))
	e.Handle(reading(10, volts(11.5)))
	if r.kinds() != "low_12v" || !strings.Contains(r.got[0].Body, "11.6 V") {
		t.Fatalf("got %s %+v", r.kinds(), r.got)
	}
	restarted := &Engine{S: s, N: r, Now: e.Now}
	restarted.Handle(reading(20, volts(11.5)))
	restarted.Handle(reading(25, volts(11.5)))
	if len(r.got) != 1 {
		t.Fatalf("a restart must not repeat today's alert: %s", r.kinds())
	}
	now = now.Add(25 * time.Hour)
	restarted.Handle(reading(1500, volts(11.5)))
	if len(r.got) != 2 {
		t.Fatalf("a day later it may alert again: %s", r.kinds())
	}
	e3, r3, _ := setup(t)
	e3.Handle(reading(0, func(st *ford.State) { st.Extra = ford.Flatten([]byte(`{"m":{"batteryStateOfCharge":{"value":40}}}`)) }))
	if r3.kinds() != "low_12v" {
		t.Fatalf("a 12 V charge under 50 percent counts too: %s", r3.kinds())
	}
	e4, r4, _ := setup(t)
	e4.Handle(reading(0, volts(12.06))) // lowish but not an alarm (the Health page flags it as 'check')
	if len(r4.got) != 0 {
		t.Fatalf("12.06 V is not an alert: %s", r4.kinds())
	}
}

func TestUnlockedAlertWaitsTenMinutesAndFiresOnce(t *testing.T) {
	e, r, _ := setup(t)
	f := false
	open := func(s *ford.State) { s.Locked = &f }
	e.Handle(reading(0, open))
	e.Handle(reading(5, open))
	if len(r.got) != 0 {
		t.Fatal("five minutes is too soon")
	}
	e.Handle(reading(11, open))
	e.Handle(reading(16, open))
	if r.kinds() != "unlocked" {
		t.Fatalf("one alert per episode: %s", r.kinds())
	}
	tr := true
	e.Handle(reading(20, func(s *ford.State) { s.Locked = &tr }))
	e.Handle(reading(25, open))
	e.Handle(reading(40, open))
	if r.kinds() != "unlocked,unlocked" {
		t.Fatalf("a new episode may alert: %s", r.kinds())
	}
	e2, r2, _ := setup(t)
	e2.Handle(reading(0, func(s *ford.State) { s.Locked, s.Ignition = &f, "ON" })) // driving: unlocked is normal
	e2.Handle(reading(30, func(s *ford.State) { s.Locked, s.Ignition = &f, "ON" }))
	if len(r2.got) != 0 {
		t.Fatalf("an unlocked car that is being driven is fine: %s", r2.kinds())
	}
}

func TestTripStartAndEndComeFromTheTripRecords(t *testing.T) {
	e, r, s := setup(t)
	e.Handle(reading(0, nil)) // baseline
	tr := &store.Trip{StartTS: t0.Unix(), Category: "work", CategorySource: "default"}
	if err := s.CreateTrip(tr); err != nil {
		t.Fatal(err)
	}
	e.Handle(reading(5, nil))
	end := t0.Add(30 * time.Minute).Unix()
	tr.EndTS, tr.DistanceKm, tr.Open = &end, 16.0934, false
	s.UpdateTrip(tr)
	e.Handle(reading(35, nil))
	if r.kinds() != "trip_start,trip_end" || !strings.Contains(r.got[1].Body, "10.0 mi (16.1 km)") || r.got[1].URL != "#/trip/1" {
		t.Fatalf("got %s %+v", r.kinds(), r.got)
	}
	// A trip that the detector threw away (an ignition blip) is not announced as ended.
	blip := &store.Trip{StartTS: t0.Add(40 * time.Minute).Unix(), Category: "work", CategorySource: "default"}
	s.CreateTrip(blip)
	e.Handle(reading(41, nil))
	s.DeleteTrip(blip.ID)
	e.Handle(reading(42, nil))
	if r.kinds() != "trip_start,trip_end,trip_start" {
		t.Fatalf("got %s", r.kinds())
	}
}

func TestSilenceFromFordIsReportedOnceAndRearms(t *testing.T) {
	e, r, _ := setup(t)
	now := t0
	last := t0.Add(-5 * time.Hour).Unix()
	linked := true
	e.Now = func() time.Time { return now }
	e.Link = func() (bool, int64) { return linked, last }
	e.Tick()
	e.Tick()
	if r.kinds() != "ford_trouble" || !strings.Contains(r.got[0].Body, "5 hours") {
		t.Fatalf("got %s %+v", r.kinds(), r.got)
	}
	last = now.Add(-10 * time.Minute).Unix() // Ford is back
	e.Tick()
	last = now.Add(-6 * time.Hour).Unix()
	e.Tick()
	if r.kinds() != "ford_trouble,ford_trouble" {
		t.Fatalf("it may alert again after recovering: %s", r.kinds())
	}
	linked = false
	last = 0
	e.silentSent = false
	e.Tick()
	if len(r.got) != 2 {
		t.Fatal("an account that is not linked has nothing to be silent about")
	}
}
