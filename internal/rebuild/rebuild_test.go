package rebuild

import (
	"strings"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/store"
	"github.com/ZachCurry13/cartrak/internal/trips"
)

func f(v float64) *float64 { return &v }

// A road trip with the car left on at a fast charger, stored as readings, and the one long wrong trip that was
// recorded from it before the detector learned that a charging car is parked.
func seed(t *testing.T) (*store.Store, *trips.Detector, time.Time) {
	s, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	t0 := time.Now().Add(-30 * time.Hour).Truncate(time.Minute)
	type step struct {
		min         int
		odo, soc    float64
		kph         float64
		kw          float64
		ign, charge string
	}
	for _, x := range []step{
		{0, 1000, 90, 100, 0, "ON", ""},
		{60, 1150, 40, 100, 0, "ON", ""},
		{70, 1150, 42, 0, 120, "ON", "CHARGING"},
		{100, 1150, 80, 0, 100, "ON", "CHARGING"},
		{110, 1151, 80, 90, 0, "ON", ""},
		{170, 1300, 35, 100, 0, "ON", ""},
		{180, 1300, 35, 0, 0, "OFF", ""},
	} {
		st := &ford.State{TS: t0.Add(time.Duration(x.min) * time.Minute), OdometerKm: f(x.odo), SoCPct: f(x.soc), SpeedKph: f(x.kph), Ignition: x.ign, ChargeState: x.charge,
			Lat: f(44), Lon: f(-93), Raw: []byte(`{}`)}
		if x.kw > 0 {
			st.ChargeKw = f(x.kw)
		}
		if _, err := s.SaveState("t", st); err != nil {
			t.Fatal(err)
		}
	}
	// The wrong long trip, with a note and a hand-chosen category the owner would not want to lose.
	e := t0.Add(180 * time.Minute).Unix()
	old := &store.Trip{StartTS: t0.Unix(), StartOdo: f(1000), StartSoC: f(90), Category: "personal", CategorySource: "default"}
	if err := s.CreateTrip(old); err != nil {
		t.Fatal(err)
	}
	old.EndTS, old.EndOdo, old.EndSoC, old.DistanceKm, old.Open = &e, f(1300), f(35), 300, false
	if err := s.UpdateTrip(old); err != nil {
		t.Fatal(err)
	}
	if err := s.SetTripCategory(old.ID, "work", "client visit"); err != nil {
		t.Fatal(err)
	}
	return s, &trips.Detector{S: s, TZ: time.UTC}, t0
}

func TestRebuildingARoadTripSplitsItAtTheChargeAndKeepsTheOwnersNotes(t *testing.T) {
	s, d, t0 := seed(t)
	p, err := Compute(s, d, t0.Add(20*time.Minute).Unix(), t0.Add(100*time.Minute).Unix(), time.Now())
	if err != nil || len(p.Blockers) > 0 {
		t.Fatalf("%v %v", err, p.Blockers)
	}
	if p.From != t0.Unix() || len(p.OldTrips) != 1 || len(p.NewTrips) != 2 {
		t.Fatalf("widened to the whole old trip, two new trips: from=%d old=%d new=%d", p.From, len(p.OldTrips), len(p.NewTrips))
	}
	if len(p.NewCharges) != 1 {
		t.Fatalf("one charge: %+v", p.NewCharges)
	}
	if err := Apply(s, d, p); err != nil {
		t.Fatal(err)
	}
	got, _ := s.ListTrips(store.TripFilter{})
	if len(got) != 2 {
		t.Fatalf("%d trips after", len(got))
	}
	for _, tr := range got {
		if tr.Open || tr.DistanceKm < 140 || tr.DistanceKm > 160 {
			t.Fatalf("each leg about 150 km: %+v", tr)
		}
		if tr.Category != "work" || tr.CategorySource != "manual" || tr.Notes != "client visit" {
			t.Fatalf("the owner's choice carries over to the legs: %+v", tr)
		}
	}
}

func TestRebuildRefusesWhenItCouldHurt(t *testing.T) {
	s, d, t0 := seed(t)
	jid := int64(7)
	s.DB.Exec(`UPDATE trips SET journey_id=?`, jid)
	p, _ := Compute(s, d, t0.Unix(), t0.Add(3*time.Hour).Unix(), time.Now())
	if len(p.Blockers) == 0 || !strings.Contains(p.Blockers[0], "journey") {
		t.Fatalf("a journey blocks it: %v", p.Blockers)
	}
	if err := Apply(s, d, p); err == nil {
		t.Fatal("apply must refuse a blocked plan")
	}
	s.DB.Exec(`UPDATE trips SET journey_id=NULL`)
	if p, _ := Compute(s, d, t0.Unix(), time.Now().Unix(), time.Now()); len(p.Blockers) == 0 {
		t.Fatal("a stretch that ends now is refused")
	}
	if p, _ := Compute(s, d, t0.Unix(), t0.Add(10*24*time.Hour).Unix(), time.Now().Add(20*24*time.Hour)); len(p.Blockers) == 0 {
		t.Fatal("more than seven days is refused")
	}
	if p, _ := Compute(s, d, t0.Add(-40*time.Hour).Unix(), t0.Add(-35*time.Hour).Unix(), time.Now()); len(p.Blockers) == 0 || !strings.Contains(p.Blockers[0], "No readings") {
		t.Fatalf("no readings that early: %v", p.Blockers)
	}
	left, _ := s.ListTrips(store.TripFilter{})
	if len(left) != 1 {
		t.Fatal("a refused rebuild changes nothing")
	}
}

// A charge that pauses and picks up again with the cable in is one session, in a rebuild as much as live. The rebuild
// once left the plug out of the readings it fed the detector, so it never merged anything.
func TestRebuildMergesAPausedChargeBecauseItKnowsThePlug(t *testing.T) {
	s, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	d := &trips.Detector{S: s, TZ: time.UTC}
	t0 := time.Now().Add(-30 * time.Hour).Truncate(time.Minute)
	for _, x := range []struct {
		min   int
		state string
		soc   float64
	}{{0, "IN_PROGRESS", 38}, {10, "IN_PROGRESS", 42}, {20, "NOT_READY", 46}, {30, "IN_PROGRESS", 47}, {40, "IN_PROGRESS", 50}, {50, "COMPLETED", 51}} {
		st := &ford.State{TS: t0.Add(time.Duration(x.min) * time.Minute), OdometerKm: f(500), SoCPct: f(x.soc), SpeedKph: f(0), Ignition: "OFF",
			ChargeState: x.state, Plug: "CONNECTED", Lat: f(44), Lon: f(-93), Raw: []byte(`{}`)}
		if x.state == "IN_PROGRESS" {
			st.ChargeKw = f(10)
		}
		if _, err := s.SaveState("t", st); err != nil {
			t.Fatal(err)
		}
	}
	p, err := Compute(s, d, t0.Unix(), t0.Add(55*time.Minute).Unix(), time.Now())
	if err != nil || len(p.Blockers) > 0 {
		t.Fatalf("%v %v", err, p.Blockers)
	}
	if len(p.NewCharges) != 1 || *p.NewCharges[0].StartSoC != 38 || *p.NewCharges[0].EndSoC != 51 {
		t.Fatalf("one session from 38 to 51: %d %+v", len(p.NewCharges), p.NewCharges)
	}
}

func TestACorrectedDistanceSurvivesARebuildOnlyForTheSameDrive(t *testing.T) {
	// split: the long wrong trip becomes two legs, so its corrected distance must not be copied onto either
	s, d, t0 := seed(t)
	_, _ = s.DB.Exec(`UPDATE trips SET distance_src='manual', distance_km=299`)
	p, _ := Compute(s, d, t0.Unix(), t0.Add(3*time.Hour).Unix(), time.Now())
	if err := Apply(s, d, p); err != nil {
		t.Fatal(err)
	}
	got, _ := s.ListTrips(store.TripFilter{})
	for _, tr := range got {
		if tr.DistanceSrc != "" || tr.DistanceKm > 200 {
			t.Fatalf("the legs are measured again: %+v", tr)
		}
	}
}

func TestRebuildingAgainLeavesWhatAlreadyMatchesAlone(t *testing.T) {
	s, d, t0 := seed(t)
	from, to := t0.Add(20*time.Minute).Unix(), t0.Add(100*time.Minute).Unix()
	p, _ := Compute(s, d, from, to, time.Now())
	if err := Apply(s, d, p); err != nil {
		t.Fatal(err)
	}
	before, _ := s.ListTrips(store.TripFilter{})
	ids := map[int64]bool{}
	for _, tr := range before {
		ids[tr.ID] = true
	}
	// everything is now as the replay would record it, so there is nothing to replace
	again, err := Compute(s, d, from, to, time.Now())
	if err != nil || len(again.Blockers) > 0 {
		t.Fatalf("%v %v", err, again.Blockers)
	}
	if len(again.OldTrips) != 0 || len(again.NewTrips) != 0 || len(again.OldCharges) != 0 || len(again.NewCharges) != 0 {
		t.Fatalf("nothing differs: old %d/%d new %d/%d", len(again.OldTrips), len(again.OldCharges), len(again.NewTrips), len(again.NewCharges))
	}
	if again.UnchangedTrips != 2 || again.UnchangedCharges != 1 {
		t.Fatalf("two trips and a charge were left alone: %d %d", again.UnchangedTrips, again.UnchangedCharges)
	}
	if err := Apply(s, d, again); err != nil {
		t.Fatal(err)
	}
	after, _ := s.ListTrips(store.TripFilter{})
	for _, tr := range after {
		if !ids[tr.ID] {
			t.Fatalf("a trip that did not change got a new id: %+v", tr)
		}
	}
}

// A charge still in progress is left out, and the stretch stops where it began, so the day's drives can be fixed while plugged in.
func TestRebuildStopsBeforeAChargeThatIsStillGoing(t *testing.T) {
	s, d, t0 := seed(t)
	start := t0.Add(200 * time.Minute).Unix()
	if err := s.CreateCharge(&store.Charge{StartTS: start, StartKwh: f(40), PeakKw: 7}); err != nil {
		t.Fatal(err)
	}
	p, err := Compute(s, d, t0.Unix(), t0.Add(4*time.Hour).Unix(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range p.Blockers {
		if strings.Contains(b, "charging session") {
			t.Fatalf("an open charge at the end must not block: %v", p.Blockers)
		}
	}
	if p.To != start-1 || p.Note == "" {
		t.Fatalf("the stretch stops before the charge: to %d want %d, note %q", p.To, start-1, p.Note)
	}
	for _, c := range p.OldCharges {
		if c.Open {
			t.Fatal("the open charge is not part of the plan")
		}
	}
	// A charge that began before the stretch and is still going leaves nothing to rebuild.
	if p, _ := Compute(s, d, t0.Add(210*time.Minute).Unix(), t0.Add(4*time.Hour).Unix(), time.Now()); len(p.Blockers) == 0 {
		t.Fatal("a stretch wholly inside an open charge is refused")
	}
}

// A rebuilt trip is classified knowing the trips of the day that stay as they are: an evening trip after the work day ended
// at Home is personal, even though the morning's trips were left alone and are not part of the plan.
func TestARebuiltTripKnowsTheDaysTripsThatWereLeftAlone(t *testing.T) {
	s, d, t0 := seed(t)
	if err := s.SaveNamedPlace(&store.NamedPlace{Name: "Home", Lat: 44, Lon: -93, RadiusM: 150}); err != nil {
		t.Fatal(err)
	}
	d.Work = func() trips.WorkDay { return trips.WorkDay{On: true, Days: 0b1111111, StaySec: 45 * 60} }
	day := t0.Add(-24 * time.Hour).Truncate(24 * time.Hour)
	kept := &store.Trip{StartTS: day.Add(8 * time.Hour).Unix(), Category: "work", CategorySource: "workday"}
	if err := s.CreateTrip(kept); err != nil {
		t.Fatal(err)
	}
	e := day.Add(15*time.Hour + 30*time.Minute).Unix()
	kept.EndTS, kept.EndLat, kept.EndLon, kept.DistanceKm, kept.Open = &e, f(44), f(-93), 30, false // back at Home
	if err := s.UpdateTrip(kept); err != nil {
		t.Fatal(err)
	}
	evening := &store.Trip{StartTS: day.Add(17 * time.Hour).Unix(), StartLat: f(44), StartLon: f(-93), Category: "work", CategorySource: "default"}
	ee := evening.StartTS + 600
	evening.EndTS, evening.DistanceKm = &ee, 4
	p := &Plan{From: day.Unix(), To: day.Add(18 * time.Hour).Unix(), NewTrips: []*store.Trip{evening}}
	if err := Apply(s, d, p); err != nil {
		t.Fatal(err)
	}
	all, _ := s.ListTrips(store.TripFilter{})
	for _, x := range all {
		if x.StartTS == evening.StartTS && x.Category != "personal" {
			t.Fatalf("the evening trip follows a work day that ended at Home: %q", x.Category)
		}
	}
}
