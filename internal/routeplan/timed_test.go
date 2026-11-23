package routeplan

import (
	"reflect"
	"testing"
)

// Five places on a line, ten minutes apart: start (0), A (1), B (2), Lunch (3), C (4). Lunch is at minute 120 for an hour; every other
// place takes 30 minutes.
func line(n int) [][]float64 {
	t := make([][]float64, n)
	for i := range t {
		t[i] = make([]float64, n)
		for j := range t[i] {
			d := float64(i - j)
			if d < 0 {
				d = -d
			}
			t[i][j] = d * 10
		}
	}
	return t
}

func visits(atLunch float64) []Visit {
	return []Visit{{0, NoTime}, {30, NoTime}, {30, NoTime}, {60, atLunch}, {30, NoTime}}
}

func TestWithNoSetTimesTheOrderIsTheLeastDriving(t *testing.T) {
	times := line(5)
	v := []Visit{{0, NoTime}, {30, NoTime}, {30, NoTime}, {30, NoTime}, {30, NoTime}}
	got, ok := BestTimed(times, v, false, false)
	if !ok || !reflect.DeepEqual(got, Best(times, false)) {
		t.Fatalf("%v, Best gives %v", got, Best(times, false))
	}
}

func TestLunchKeepsItsTimeAndTheOthersFitAroundIt(t *testing.T) {
	times := line(5)
	// Lunch at minute 40: only A fits before it (A at 10, left at 40, lunch reached at 60 is 20 late), so lunch comes first, waits.
	got, ok := BestTimed(times, visits(40), false, false)
	if !ok || got[1] != 3 {
		t.Fatalf("lunch first: %v", got)
	}
	// Lunch at minute 120: two of the others fit before it.
	got, ok = BestTimed(times, visits(120), false, false)
	s := Schedule(times, visits(120), got, false)
	var lunch Arrival
	for i, k := range got[1:] {
		if k == 3 {
			lunch = s[i]
		}
	}
	if !ok || lunch.LateMin > OnTimeSlack || lunch.ArriveMin+lunch.WaitMin != 120 {
		t.Fatalf("lunch at 120: order %v, %+v", got, lunch)
	}
}

func TestAnImpossibleTimeGivesTheLeastLateOrderAndSaysSo(t *testing.T) {
	times := line(5)
	got, ok := BestTimed(times, visits(5), false, false) // lunch is 30 minutes away: five minutes cannot be made
	if ok || got == nil || got[1] != 3 {
		t.Fatalf("not on time, but lunch as soon as it can be: %v %v", got, ok)
	}
}

func TestTheLastPlaceStaysLastWhenAsked(t *testing.T) {
	times := line(5)
	v := []Visit{{0, NoTime}, {30, NoTime}, {30, NoTime}, {30, NoTime}, {30, NoTime}}
	times[0][4], times[4][0] = 1, 1 // place 4 is right next to the start: the quickest day would visit it first
	got, _ := BestTimed(times, v, false, true)
	if got[len(got)-1] != 4 {
		t.Fatalf("Costco stays last: %v", got)
	}
	free, _ := BestTimed(times, v, false, false)
	if free[len(free)-1] == 4 {
		t.Fatalf("without the tick it may move: %v", free)
	}
}

func TestScheduleWaitsForASetTimeAndCountsTheWayBack(t *testing.T) {
	times := line(3)
	s := Schedule(times, []Visit{{0, NoTime}, {15, 60}, {30, NoTime}}, []int{0, 1, 2}, true)
	if s[0].ArriveMin != 10 || s[0].WaitMin != 50 || s[0].LeaveMin != 75 || s[1].ArriveMin != 85 || s[1].LeaveMin != 115 || s[2].ArriveMin != 135 {
		t.Fatalf("%+v", s)
	}
}
