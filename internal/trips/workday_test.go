package trips

import (
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

var (
	homeAt   = []spot{{45, -93, 150}}
	workDays = WorkDay{On: true, Days: 0b0111110, StaySec: 45 * 60} // Monday to Friday
	tuesday  = time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC)
	saturday = time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)
	awayLat  = 45.1
	homeLat  = 45.0
	homeLon  = -93.0
	awayLon  = -93.2
)

// trip is a finished trip that starts and ends at the given hours of a day, ending at Home or away.
func trip(day time.Time, from, to float64, athome bool, cat string) *store.Trip {
	s, e := day.Add(time.Duration(from*60)*time.Minute).Unix(), day.Add(time.Duration(to*60)*time.Minute).Unix()
	lat, lon := awayLat, awayLon
	if athome {
		lat, lon = homeLat, homeLon
	}
	return &store.Trip{StartTS: s, EndTS: &e, EndLat: &lat, EndLon: &lon, Category: cat}
}

func at(day time.Time, hour float64) int64 {
	return day.Add(time.Duration(hour*60) * time.Minute).Unix()
}

func TestTheFirstOutingOfAWorkDayIsWorkWhateverTheClockSays(t *testing.T) {
	for _, h := range []float64{5, 8, 13, 21.5} {
		if c, ok := workDayCategory(workDays, homeAt, time.UTC, nil, at(tuesday, h)); !ok || c != Work {
			t.Fatalf("the first outing at %v is work: %q %v", h, c, ok)
		}
	}
}

func TestLaterOutingsAfterGettingHomeArePersonal(t *testing.T) {
	prior := []*store.Trip{trip(tuesday, 8, 15.5, true, Work)}
	if c, ok := workDayCategory(workDays, homeAt, time.UTC, prior, at(tuesday, 16.75)); !ok || c != Personal {
		t.Fatalf("a pizza run after the work day is personal: %q %v", c, ok)
	}
}

func TestAStopAwayFromHomeKeepsTheOutingsCategory(t *testing.T) {
	prior := []*store.Trip{trip(tuesday, 8, 9, false, Work), trip(tuesday, 9.5, 10, false, Work)}
	if c, ok := workDayCategory(workDays, homeAt, time.UTC, prior, at(tuesday, 11)); !ok || c != Work {
		t.Fatalf("a drive on from a customer is the same outing: %q %v", c, ok)
	}
	// also on a Saturday, so a hand-labeled work outing carries its later legs
	prior = []*store.Trip{trip(saturday, 18, 19, false, Work)}
	if c, ok := workDayCategory(workDays, homeAt, time.UTC, prior, at(saturday, 22)); !ok || c != Work {
		t.Fatalf("a weekend work outing carries on: %q %v", c, ok)
	}
	// but a night away is a new outing
	prior = []*store.Trip{trip(tuesday, 15, 16, false, Work)}
	if c, ok := workDayCategory(workDays, homeAt, time.UTC, prior, at(tuesday.AddDate(0, 0, 1), 8)); !ok || c != Work {
		t.Fatalf("the next morning is the first outing of its own day: %q %v", c, ok)
	}
}

func TestALunchAtHomeDoesNotEndTheWorkDay(t *testing.T) {
	prior := []*store.Trip{trip(tuesday, 8, 12, true, Work)}
	if c, _ := workDayCategory(workDays, homeAt, time.UTC, prior, at(tuesday, 12.5)); c != Work {
		t.Fatalf("back out after 30 minutes is still work: %q", c)
	}
	if c, _ := workDayCategory(workDays, homeAt, time.UTC, prior, at(tuesday, 13)); c != Personal {
		t.Fatalf("an hour at home ends the work day: %q", c)
	}
}

func TestTheRuleStaysQuietWhenItCannotJudge(t *testing.T) {
	prior := []*store.Trip{trip(tuesday, 8, 9, true, Work)}
	if _, ok := workDayCategory(workDays, homeAt, time.UTC, nil, at(saturday, 10)); ok {
		t.Fatal("a weekend outing is left to the rules")
	}
	if _, ok := workDayCategory(WorkDay{}, homeAt, time.UTC, nil, at(tuesday, 8)); ok {
		t.Fatal("off means off")
	}
	if _, ok := workDayCategory(workDays, nil, time.UTC, nil, at(tuesday, 8)); ok {
		t.Fatal("without a Home place it cannot tell an outing from the rest")
	}
	blank := trip(tuesday, 8, 9, false, Work)
	blank.EndLat, blank.EndLon = nil, nil
	if _, ok := workDayCategory(workDays, homeAt, time.UTC, []*store.Trip{blank}, at(tuesday, 10)); ok {
		t.Fatal("a trip with no end position leaves it unsure")
	}
	_ = prior
}

func TestSecondHomeCountsAsHome(t *testing.T) {
	for n, want := range map[string]bool{"Home": true, " home ": true, "Home 2": true, "Home2": false, "Home office": false, "Homestead": false} {
		if isHomeName(n) != want {
			t.Fatalf("%q: %v", n, !want)
		}
	}
}
