package alerts

import (
	"strings"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

func daysAt(t *testing.T, s *store.Store, now time.Time, highs ...float64) { // highs[0] is yesterday, then the day before, and so on
	t.Helper()
	for i, c := range highs {
		ts := now.AddDate(0, 0, -(i + 1)).Add(-2 * time.Hour).Unix()
		if _, err := s.DB.Exec(`INSERT INTO samples(ts,outside_c) VALUES(?,?)`, ts, c); err != nil {
			t.Fatal(err)
		}
	}
}

func TestWinterTiresAreSuggestedAfterThreeColdDaysOncePerYearAndOnlyForALoggedCar(t *testing.T) {
	e, r, s := setup(t)
	now := time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)
	e.Now = func() time.Time { return now }
	daysAt(t, s, now, 3, 2, 4)
	e.checkTires()
	if len(r.got) != 0 {
		t.Fatalf("nobody logged tires: no opinion: %v", r.kinds())
	}
	s.AddTireSet(store.TireSet{Name: "Summer", Kind: "summer", FromTS: now.AddDate(0, -6, 0).Unix()})
	now = now.Add(24 * time.Hour) // the next day's check
	e.checkTires()
	if r.kinds() != "tire_swap" || !strings.Contains(r.got[0].Body, "45°F") || r.got[0].URL != "#/tires" {
		t.Fatalf("three cold days with summer tires on: %v %v", r.kinds(), r.got)
	}
	now = now.Add(24 * time.Hour)
	e.checkTires()
	if len(r.got) != 1 {
		t.Fatalf("once a year: %v", r.kinds())
	}
}

func TestNoWinterReminderWhenTheDaysAreNotAllColdOrWinterTiresAreAlreadyOn(t *testing.T) {
	e, r, s := setup(t)
	now := time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)
	e.Now = func() time.Time { return now }
	s.AddTireSet(store.TireSet{Name: "Summer", Kind: "summer", FromTS: now.AddDate(0, -6, 0).Unix()})
	daysAt(t, s, now, 3, 12, 4) // one mild day in the three
	e.checkTires()
	if len(r.got) != 0 {
		t.Fatalf("a mild day breaks the run: %v", r.kinds())
	}
	s.AddTireSet(store.TireSet{Name: "Winter", Kind: "winter", FromTS: now.AddDate(0, 0, -20).Unix()})
	s.DB.Exec(`DELETE FROM samples`)
	daysAt(t, s, now, 1, 2, 0)
	now = now.Add(24 * time.Hour)
	e.checkTires()
	if len(r.got) != 0 {
		t.Fatalf("winter tires are already on: %v", r.kinds())
	}
}

func TestTheSummerSetIsSuggestedBackInSpringOnceItStaysMild(t *testing.T) {
	e, r, s := setup(t)
	now := time.Date(2027, 4, 20, 9, 0, 0, 0, time.UTC)
	e.Now = func() time.Time { return now }
	s.AddTireSet(store.TireSet{Name: "Winter", Kind: "winter", FromTS: now.AddDate(0, -5, 0).Unix()})
	daysAt(t, s, now, 14, 15, 13, 16, 18)
	e.checkTires()
	if r.kinds() != "tire_swap" || !strings.Contains(r.got[0].Title, "winter tires off") {
		t.Fatalf("five mild days with winter tires on in April: %v %v", r.kinds(), r.got)
	}
}
