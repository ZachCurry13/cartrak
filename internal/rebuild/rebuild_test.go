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
