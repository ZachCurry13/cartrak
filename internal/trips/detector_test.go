package trips

import (
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/store"
)

func reading(ts time.Time, ign string, odo, soc float64) *ford.State {
	return &ford.State{TS: ts, Ignition: ign, OdometerKm: &odo, SoCPct: &soc}
}

func TestTripLifecycleAndRule(t *testing.T) {
	s, err := store.Open(t.TempDir())
	t.Cleanup(func() { s.Close() })
	if err != nil {
		t.Fatal(err)
	}
	s.SaveRule(store.Rule{Name: "Work hours", Category: Work, Enabled: true, Days: 62, StartMin: 480, EndMin: 1020})
	d := &Detector{S: s, TZ: time.UTC}
	t0 := time.Date(2026, 10, 12, 8, 30, 0, 0, time.UTC) // Monday 8:30

	steps := []*ford.State{
		reading(t0, "OFF", 1000, 80),
		reading(t0.Add(1*time.Minute), "ON", 1000, 80),
		reading(t0.Add(10*time.Minute), "ON", 1012, 76),
		reading(t0.Add(20*time.Minute), "OFF", 1020, 74),
	}
	for _, st := range steps {
		if err := d.Process(st); err != nil {
			t.Fatal(err)
		}
	}
	trips, _ := s.ListTrips(store.TripFilter{})
	if len(trips) != 1 {
		t.Fatalf("want 1 trip, got %d", len(trips))
	}
	tr := trips[0]
	if tr.Open || tr.DistanceKm != 20 || tr.Category != Work || tr.CategorySource != "rule" {
		t.Fatalf("bad trip: %+v", tr)
	}

	// A manual choice survives re-running rules.
	s.SetTripCategory(tr.ID, Personal, "dentist")
	if err := d.Reclassify(); err != nil {
		t.Fatal(err)
	}
	tr, _ = s.GetTrip(tr.ID)
	if tr.Category != Personal || tr.CategorySource != "manual" {
		t.Fatalf("manual override lost: %+v", tr)
	}
}

func TestIgnitionBlipIsDiscarded(t *testing.T) {
	s, _ := store.Open(t.TempDir())
	t.Cleanup(func() { s.Close() })
	d := &Detector{S: s, TZ: time.UTC}
	t0 := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	d.Process(reading(t0, "ON", 50, 50))
	d.Process(reading(t0.Add(30*time.Second), "OFF", 50, 50))
	if ts, _ := s.ListTrips(store.TripFilter{}); len(ts) != 0 {
		t.Fatalf("blip kept: %d", len(ts))
	}
}

func TestChargeSession(t *testing.T) {
	s, _ := store.Open(t.TempDir())
	t.Cleanup(func() { s.Close() })
	d := &Detector{S: s, TZ: time.UTC}
	t0 := time.Now()
	kw := 11.0
	a := reading(t0, "OFF", 10, 40)
	a.ChargeState = "ChargingAC"
	a.ChargeKw = &kw
	b := reading(t0.Add(time.Hour), "OFF", 10, 70)
	b.ChargeState = "ChargeCompleted"
	d.Process(a)
	d.Process(b)
	cs, _ := s.ListCharges(10)
	if len(cs) != 1 || cs[0].Open || *cs[0].EndSoC != 70 || cs[0].PeakKw != 11 {
		t.Fatalf("bad charge: %+v", cs)
	}
}

// The same instant is 8:30 in Minnesota and 9:30 in New York, so a "9 to 5" work
// rule must treat the two trips differently.
func TestRulesUseTheTimeZoneWhereTheTripStarts(t *testing.T) {
	s, _ := store.Open(t.TempDir())
	t.Cleanup(func() { s.Close() })
	s.SaveRule(store.Rule{Name: "9 to 5", Category: Work, Enabled: true, Days: 62, StartMin: 9 * 60, EndMin: 17 * 60})
	d := &Detector{S: s, TZ: time.UTC} // default zone is UTC; locations decide
	at := time.Date(2026, 10, 14, 13, 30, 0, 0, time.UTC)

	drive := func(lat, lon float64, ts time.Time) *store.Trip {
		st := reading(ts, "ON", 100, 80)
		st.Lat, st.Lon = &lat, &lon
		if err := d.Process(st); err != nil {
			t.Fatal(err)
		}
		off := reading(ts.Add(10*time.Minute), "OFF", 110, 78)
		off.Lat, off.Lon = &lat, &lon
		d.Process(off)
		ts2, _ := s.ListTrips(store.TripFilter{Limit: 1})
		return ts2[0]
	}
	mn := drive(44.9778, -93.2650, at)                  // Minneapolis
	ny := drive(40.7128, -74.0060, at.Add(2*time.Hour)) // New York, an hour later by the clock there
	if mn.TZ != "America/Chicago" || mn.Category != Personal {
		t.Fatalf("Minnesota 8:30 should be personal in America/Chicago: %+v", mn)
	}
	if ny.TZ != "America/New_York" || ny.Category != Work {
		t.Fatalf("New York 10:30 should be work in America/New_York: %+v", ny)
	}

	// Without a location the default zone is used.
	noGPS := reading(at.Add(48*time.Hour), "ON", 200, 60)
	d.Process(noGPS)
	d.Process(reading(at.Add(48*time.Hour+10*time.Minute), "OFF", 210, 58))
	last, _ := s.ListTrips(store.TripFilter{Limit: 1})
	if last[0].TZ != "UTC" {
		t.Fatalf("no location should fall back to the default zone, got %q", last[0].TZ)
	}

	// Re-applying rules keeps judging each trip by its own zone.
	if err := d.Reclassify(); err != nil {
		t.Fatal(err)
	}
	mn2, _ := s.GetTrip(mn.ID)
	ny2, _ := s.GetTrip(ny.ID)
	if mn2.Category != Personal || ny2.Category != Work {
		t.Fatalf("reclassify changed the result: mn=%s ny=%s", mn2.Category, ny2.Category)
	}
}

func TestACarLeftOnAtAFastChargerSplitsTheRoadTripIntoTwoTrips(t *testing.T) {
	s, err := store.Open(t.TempDir())
	t.Cleanup(func() { s.Close() })
	if err != nil {
		t.Fatal(err)
	}
	d := &Detector{S: s, TZ: time.UTC}
	t0 := time.Date(2026, 10, 18, 9, 0, 0, 0, time.UTC)
	speed := func(st *ford.State, kph float64) *ford.State { st.SpeedKph = &kph; return st }
	charge := func(st *ford.State) *ford.State {
		kw := 120.0
		st.ChargeKw, st.ChargeState = &kw, "CHARGING"
		return speed(st, 0)
	}
	steps := []*ford.State{
		speed(reading(t0, "ON", 1000, 90), 100),
		speed(reading(t0.Add(60*time.Minute), "ON", 1150, 40), 100),
		charge(reading(t0.Add(70*time.Minute), "ON", 1150, 42)), // arrives, ignition still on, charging
		charge(reading(t0.Add(100*time.Minute), "ON", 1150, 80)),
		speed(reading(t0.Add(110*time.Minute), "ON", 1151, 80), 90), // leaves
		speed(reading(t0.Add(170*time.Minute), "ON", 1300, 35), 100),
		reading(t0.Add(180*time.Minute), "OFF", 1300, 35),
	}
	for _, st := range steps {
		if err := d.Process(st); err != nil {
			t.Fatal(err)
		}
	}
	ts, _ := s.ListTrips(store.TripFilter{})
	if len(ts) != 2 {
		t.Fatalf("want two drives either side of the charge, got %d", len(ts))
	}
	for _, tr := range ts {
		if tr.Open {
			t.Fatalf("both should be finished: %+v", tr)
		}
		if tr.DistanceKm < 140 || tr.DistanceKm > 160 {
			t.Fatalf("each leg is about 150 km: %+v", tr.DistanceKm)
		}
	}
}

// The road trip of 2026-10-18 as the car reported it: at each fast charger the charge status was IN_PROGRESS with
// 0.0 kW and the ignition left on, which used to leave the whole day as one long trip with no charging sessions.
func TestFastChargerStopsWithZeroPowerSplitTheTripAndMakeChargingSessions(t *testing.T) {
	s, err := store.Open(t.TempDir())
	t.Cleanup(func() { s.Close() })
	if err != nil {
		t.Fatal(err)
	}
	d := &Detector{S: s, TZ: time.UTC}
	t0 := time.Date(2026, 10, 18, 9, 0, 0, 0, time.UTC)
	step := func(min int, odo, soc, kph float64, charge string) {
		st := reading(t0.Add(time.Duration(min)*time.Minute), "ON", odo, soc)
		st.SpeedKph = &kph
		st.ChargeState = charge
		st.Plug = "CONNECTED"
		if charge == "" {
			st.Plug = "DISCONNECTED"
		}
		kw := 0.0
		st.ChargeKw = &kw
		if err := d.Process(st); err != nil {
			t.Fatal(err)
		}
	}
	step(0, 1000, 90, 100, "")
	step(60, 1150, 40, 100, "")
	step(70, 1150, 40, 0, "IN_PROGRESS")
	step(100, 1150, 80, 0, "IN_PROGRESS")
	step(101, 1150, 80, 0, "NOT_READY")
	step(110, 1151, 80, 90, "")
	step(170, 1300, 35, 100, "")
	step(180, 1300, 35, 0, "")
	ts, _ := s.ListTrips(store.TripFilter{})
	cs, _ := s.ListCharges(10)
	if len(ts) != 2 || len(cs) != 1 {
		t.Fatalf("two drives and one charging session, got %d trips and %d sessions", len(ts), len(cs))
	}
	if cs[0].Open || cs[0].EndSoC == nil || *cs[0].EndSoC != 80 {
		t.Fatalf("the session ended at 80%%: %+v", cs[0])
	}
}

func TestSittingInTheCarWithTheIgnitionOnIsNotATrip(t *testing.T) {
	s, err := store.Open(t.TempDir())
	t.Cleanup(func() { s.Close() })
	if err != nil {
		t.Fatal(err)
	}
	d := &Detector{S: s, TZ: time.UTC}
	t0 := time.Date(2026, 10, 19, 9, 0, 0, 0, time.UTC)
	still := func(min int) *ford.State {
		st := reading(t0.Add(time.Duration(min)*time.Minute), "ON", 5000, 70)
		kph := 0.0
		st.SpeedKph = &kph
		return st
	}
	for _, min := range []int{0, 5, 10, 20, 30, 40} { // 40 minutes with the radio on
		if err := d.Process(still(min)); err != nil {
			t.Fatal(err)
		}
	}
	d.Process(reading(t0.Add(41*time.Minute), "OFF", 5000, 70))
	if ts, _ := s.ListTrips(store.TripFilter{}); len(ts) != 0 {
		t.Fatalf("no movement, no trip: %d", len(ts))
	}
	// Then a real drive starts when the car moves.
	drive := reading(t0.Add(50*time.Minute), "ON", 5000.1, 70)
	kph := 40.0
	drive.SpeedKph = &kph
	d.Process(still(49))
	d.Process(drive)
	far := reading(t0.Add(70*time.Minute), "ON", 5020, 66)
	far.SpeedKph = &kph
	d.Process(far)
	d.Process(reading(t0.Add(71*time.Minute), "OFF", 5020, 66))
	ts, _ := s.ListTrips(store.TripFilter{})
	if len(ts) != 1 || ts[0].DistanceKm < 19 {
		t.Fatalf("one real trip of about 20 km: %+v", ts)
	}
}

func TestASessionKeepsTheChargerTypeTheCarReportedWhileCharging(t *testing.T) {
	s, _ := store.Open(t.TempDir())
	t.Cleanup(func() { s.Close() })
	d := &Detector{S: s, TZ: time.UTC}
	t0 := time.Now()
	kw := 45.0
	a := reading(t0, "OFF", 10, 40)
	a.ChargeState, a.ChargeKw, a.StationType = "IN_PROGRESS", &kw, "DC"
	b := reading(t0.Add(30*time.Minute), "OFF", 10, 70)
	b.ChargeState, b.StationType = "NOT_READY", "DC" // the field keeps its last value after the stop
	d.Process(a)
	d.Process(b)
	cs, _ := s.ListCharges(10)
	if len(cs) != 1 || cs[0].Station != "DC" {
		t.Fatalf("station: %+v", cs)
	}
	// A parked reading with a stale station type must not start or colour anything.
	if stationOf(b) != "" {
		t.Fatal("stale station type is ignored when not charging")
	}
}

func TestARemoteStartIsNotATripEvenWhenTheCarSendsNoSpeed(t *testing.T) {
	s, _ := store.Open(t.TempDir())
	t.Cleanup(func() { s.Close() })
	d := &Detector{S: s, TZ: time.UTC}
	t0 := time.Now()
	a := reading(t0, "ON", 100, 80)
	a.SpeedKph = nil
	a.Extra = map[string]any{"remoteStartCountdownTimer.value": 540.0}
	b := reading(t0.Add(5*time.Minute), "ON", 100, 80)
	b.SpeedKph = nil
	b.Extra = map[string]any{"remoteStartCountdownTimer.value": 240.0}
	d.Process(a)
	d.Process(b)
	if ts, _ := s.ListTrips(store.TripFilter{}); len(ts) != 0 {
		t.Fatalf("a remote start made a trip: %+v", ts)
	}
	c := reading(t0.Add(8*time.Minute), "ON", 100, 80)
	c.SpeedKph = nil
	d.Process(c) // no countdown: someone is driving and the car sends no speed
	if ts, _ := s.ListTrips(store.TripFilter{}); len(ts) != 1 {
		t.Fatalf("driving without a speed field still counts: %+v", ts)
	}
}

func TestAWakeReadingWithNoSpeedDoesNotStartATripUntilTheOdometerMoves(t *testing.T) {
	s, _ := store.Open(t.TempDir())
	t.Cleanup(func() { s.Close() })
	d := &Detector{S: s, TZ: time.UTC}
	t0 := time.Now()
	sp := 0.0
	parked := reading(t0, "OFF", 500, 60)
	parked.SpeedKph = &sp
	d.Process(parked) // the feed does send a speed
	wake := reading(t0.Add(10*time.Minute), "ON", 500, 60)
	d.Process(wake) // ignition on, no speed, the odometer has not moved
	if ts, _ := s.ListTrips(store.TripFilter{}); len(ts) != 0 {
		t.Fatalf("waking is not a trip: %+v", ts)
	}
	went := reading(t0.Add(11*time.Minute), "ON", 500.4, 60)
	d.Process(went) // no speed, but the odometer moved
	if ts, _ := s.ListTrips(store.TripFilter{}); len(ts) != 1 {
		t.Fatalf("a car that has moved starts a trip: %+v", ts)
	}
}

func chargeReading(ts time.Time, state, plug string, soc float64) *ford.State {
	st := reading(ts, "OFF", 100, soc)
	st.ChargeState, st.Plug = state, plug
	lat, lon := 44.76, -85.62
	st.Lat, st.Lon = &lat, &lon
	return st
}

func TestAPausedChargeThatResumesIsOneSessionButAnUnpluggedOneIsNot(t *testing.T) {
	s, _ := store.Open(t.TempDir())
	t.Cleanup(func() { s.Close() })
	d := &Detector{S: s, TZ: time.UTC}
	t0 := time.Now().Add(-3 * time.Hour)
	at := func(min int) time.Time { return t0.Add(time.Duration(min) * time.Minute) }
	d.Process(chargeReading(at(0), "IN_PROGRESS", "CONNECTED", 38))
	d.Process(chargeReading(at(39), "IN_PROGRESS", "CONNECTED", 46))
	d.Process(chargeReading(at(40), "NOT_READY", "CONNECTED", 46)) // a pause
	if cs, _ := s.ListCharges(10); len(cs) != 1 || cs[0].Open {
		t.Fatalf("paused, so the session is closed for now: %+v", cs)
	}
	d.Process(chargeReading(at(50), "IN_PROGRESS", "CONNECTED", 48)) // resumed ten minutes later
	d.Process(chargeReading(at(52), "FAULT", "CONNECTED", 48))
	cs, _ := s.ListCharges(10)
	if len(cs) != 1 || *cs[0].EndSoC != 48 || cs[0].StartSoC == nil || *cs[0].StartSoC != 38 {
		t.Fatalf("one session from 38 to 48: %+v", cs)
	}
	// Unplugged and plugged in again is a new session.
	d.Process(chargeReading(at(60), "NOT_READY", "DISCONNECTED", 48))
	d.Process(chargeReading(at(62), "IN_PROGRESS", "CONNECTED", 48))
	if cs, _ := s.ListCharges(10); len(cs) != 2 {
		t.Fatalf("a new cable is a new session: %d", len(cs))
	}
	// A long gap is a new session too.
	d.Process(chargeReading(at(70), "NOT_READY", "CONNECTED", 50))
	d.Process(chargeReading(at(100), "IN_PROGRESS", "CONNECTED", 50))
	if cs, _ := s.ListCharges(10); len(cs) != 3 {
		t.Fatalf("half an hour later is a new session: %d", len(cs))
	}
}

func TestAQuickReplugAtAChargerIsOneSessionButASlowOneIsNot(t *testing.T) {
	s, _ := store.Open(t.TempDir())
	t.Cleanup(func() { s.Close() })
	d := &Detector{S: s, TZ: time.UTC}
	t0 := time.Now().Add(-3 * time.Hour)
	at := func(sec int) time.Time { return t0.Add(time.Duration(sec) * time.Second) }
	d.Process(chargeReading(at(0), "IN_PROGRESS", "CONNECTED", 53))
	d.Process(chargeReading(at(600), "IN_PROGRESS", "CONNECTED", 68))
	d.Process(chargeReading(at(630), "PAUSED", "CONNECTED", 68))       // the charger stops
	d.Process(chargeReading(at(750), "NOT_READY", "DISCONNECTED", 68)) // the plug comes out
	d.Process(chargeReading(at(780), "NOT_READY", "CONNECTED", 68))    // and goes back in
	d.Process(chargeReading(at(810), "IN_PROGRESS", "CONNECTED", 68))
	d.Process(chargeReading(at(1200), "IN_PROGRESS", "CONNECTED", 82))
	cs, _ := s.ListCharges(10)
	if len(cs) != 1 || *cs[0].StartSoC != 53 {
		t.Fatalf("a replug three minutes later is the same stop: %+v", cs)
	}
	// out for ten minutes is a different stop
	d.Process(chargeReading(at(1230), "PAUSED", "CONNECTED", 82))
	d.Process(chargeReading(at(1300), "NOT_READY", "DISCONNECTED", 82))
	d.Process(chargeReading(at(1900), "IN_PROGRESS", "CONNECTED", 82))
	if cs, _ := s.ListCharges(10); len(cs) != 2 {
		t.Fatalf("ten minutes is a new session: %d", len(cs))
	}
}

func TestTheReportedPowerIsAddedUpAcrossASession(t *testing.T) {
	s, _ := store.Open(t.TempDir())
	t.Cleanup(func() { s.Close() })
	d := &Detector{S: s, TZ: time.UTC}
	t0 := time.Now().Add(-3 * time.Hour)
	kw := func(min int, v float64) *ford.State {
		st := chargeReading(t0.Add(time.Duration(min)*time.Minute), "IN_PROGRESS", "CONNECTED", 50)
		st.ChargeKw = &v
		return st
	}
	d.Process(kw(0, 10))  // the session starts; nothing is counted before the next reading
	d.Process(kw(10, 10)) // 10 minutes at 10 kW = 1.667 kWh
	d.Process(kw(20, 4))  // 10 minutes from 10 down to 4 kW = 1.167 kWh
	cs, _ := s.ListCharges(5)
	if len(cs) != 1 || cs[0].PowerKwh == nil || *cs[0].PowerKwh < 2.83 || *cs[0].PowerKwh > 2.84 {
		t.Fatalf("expected 2.83 kWh from the power: %+v", cs[0].PowerKwh)
	}
	// a reading hours later does not add the silence in between
	d.Process(kw(300, 4))
	cs, _ = s.ListCharges(5)
	if *cs[0].PowerKwh > 2.84 {
		t.Fatalf("a long silence must not be counted: %v", *cs[0].PowerKwh)
	}
}

func TestAModeGivesEveryTripItsCategory(t *testing.T) {
	s, _ := store.Open(t.TempDir())
	t.Cleanup(func() { s.Close() })
	d := &Detector{S: s, TZ: time.UTC}
	day := time.Now().Format("2006-01-02")
	_ = s.AddTimeOff(&store.TimeOff{From: day, To: day, Kind: "medical"})
	cat, _, src := d.classify(time.Now().Unix(), nil, nil, time.Local)
	if cat != "medical" || src != "mode" {
		t.Fatalf("medical mode: %s %s", cat, src)
	}
	_ = s.AddTimeOff(&store.TimeOff{From: day, To: day, Kind: "vacation"})
	if cat, _, src = d.classify(time.Now().Unix(), nil, nil, time.Local); cat != "personal" || src != "vacation" {
		t.Fatalf("the newest mode wins: %s %s", cat, src)
	}
}
