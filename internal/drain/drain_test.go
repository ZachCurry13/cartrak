package drain

import (
	"math"
	"testing"

	"github.com/ZachCurry13/cartrak/internal/store"
)

func pt(h float64, soc float64, odo float64) store.TimePoint {
	return store.TimePoint{TS: int64(h * 3600), SoC: &soc, Odo: &odo}
}

func TestAParkedStayLosesItsDropOverItsHours(t *testing.T) {
	// 48 hours parked, 80 down to 78: 1 percent a day.
	pts := []store.TimePoint{pt(0, 80, 100), pt(12, 79.5, 100), pt(24, 79, 100), pt(48, 78, 100)}
	got := Stretches(pts)
	if len(got) != 1 || got[0].Hours != 48 || math.Abs(got[0].DropPct-2) > 1e-9 {
		t.Fatalf("%+v", got)
	}
	rep := Analyze(pts, 48*3600)
	if !rep.Week.Enough || math.Abs(rep.Week.PctPerDay-1) > 1e-9 {
		t.Fatalf("1 percent a day: %+v", rep.Week)
	}
}

func TestDrivingChargingAndUnflaggedMovementEndAStay(t *testing.T) {
	driving := pt(10, 70, 105)
	driving.Driving = true
	charging := pt(30, 60, 105)
	charging.Charging = true
	pts := []store.TimePoint{
		pt(0, 80, 100), pt(8, 79, 100), // 8 h parked
		driving,
		pt(12, 70, 105), pt(20, 69, 105), // 8 h parked again
		charging,
		pt(40, 60, 105), pt(42, 60, 110), // odometer moved: not parked, and too short anyway
		pt(43, 60, 110), pt(45, 75, 110), // a rise of 15 points: charging nobody flagged ends the stay
	}
	got := Stretches(pts)
	if len(got) != 2 || got[0].DropPct != 1 || got[1].DropPct != 1 {
		t.Fatalf("two 8 hour stays, each losing 1: %+v", got)
	}
}

func TestTooLittleParkedTimeSaysSoAndOddDropsAreIgnored(t *testing.T) {
	short := []store.TimePoint{pt(0, 80, 1), pt(2, 79, 1)}
	if len(Stretches(short)) != 0 || Analyze(short, 3600).Week.Enough {
		t.Fatal("a two hour stop is not enough")
	}
	odd := []store.TimePoint{pt(0, 90, 1), pt(10, 40, 1)}
	if len(Stretches(odd)) != 0 {
		t.Fatal("a 50 point fall while parked is a data oddity")
	}
	r := Analyze([]store.TimePoint{pt(0, 80, 1), pt(5, 80, 1)}, 5*3600)
	if r.Week.Enough {
		t.Fatalf("five hours is not enough for a daily rate: %+v", r.Week)
	}
}
