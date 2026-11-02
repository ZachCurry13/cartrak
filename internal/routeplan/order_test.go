package routeplan

import (
	"math"
	"testing"
)

// lineTimes puts places on a line, one position each in minutes, so the time between two is the gap between them.
func lineTimes(pos ...float64) [][]float64 {
	t := make([][]float64, len(pos))
	for i := range t {
		t[i] = make([]float64, len(pos))
		for j := range t[i] {
			t[i][j] = math.Abs(pos[i] - pos[j])
		}
	}
	return t
}

func TestBestVisitsPlacesInTheShortestOrderAndSavesTimeOverTheGivenOrder(t *testing.T) {
	times := lineTimes(0, 30, 10, 40, 20) // the given order zigzags
	got := Best(times, true)
	if len(got) != 5 || got[0] != 0 {
		t.Fatalf("%v", got)
	}
	if m := TotalMinutes(times, got, true); m != 80 {
		t.Fatalf("out to 40 and back is 80 minutes, got %v with %v", m, got)
	}
	if TotalMinutes(times, []int{0, 1, 2, 3, 4}, true) <= 80 {
		t.Fatal("the given order is longer")
	}
	open := Best(times, false)
	if m := TotalMinutes(times, open, false); m != 40 || open[len(open)-1] != 3 {
		t.Fatalf("without coming back it ends at the far end: %v %v", open, m)
	}
}

func TestBestHandlesTinyAndFullSizedLists(t *testing.T) {
	if g := Best(lineTimes(0), true); len(g) != 1 || g[0] != 0 {
		t.Fatalf("%v", g)
	}
	if g := Best(lineTimes(0, 5), true); len(g) != 2 || g[1] != 1 {
		t.Fatalf("%v", g)
	}
	pos := []float64{0}
	for i := 1; i <= 10; i++ { // ten places, scrambled along a line
		pos = append(pos, float64((i*7)%11)*3)
	}
	times := lineTimes(pos...)
	g := Best(times, true)
	seen := map[int]bool{}
	for _, p := range g {
		seen[p] = true
	}
	if len(g) != 11 || len(seen) != 11 || g[0] != 0 {
		t.Fatalf("every place once: %v", g)
	}
	far := 0.0
	for _, p := range pos {
		far = math.Max(far, p)
	}
	if m := TotalMinutes(times, g, true); m != 2*far {
		t.Fatalf("on a line the best round trip is out to the far end and back (%v), got %v", 2*far, m)
	}
}
