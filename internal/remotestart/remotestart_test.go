package remotestart

import "testing"

func TestRemoteStartsAreCountedWhenTheCountdownRises(t *testing.T) {
	pts := []Point{{1, 0}, {2, 600}, {3, 540}, {4, 120}, {5, 0}, {6, 0}, {7, 900}, {8, 850}, {9, 0}}
	if got := Count(pts); got != 2 {
		t.Fatalf("two separate starts: %d", got)
	}
	if got := Count([]Point{{1, 300}, {2, 280}, {3, 0}}); got != 1 {
		t.Fatalf("one start seen from the middle: %d", got)
	}
	if got := Count([]Point{{1, 100}, {2, 900}, {3, 880}}); got != 2 {
		t.Fatalf("a countdown that jumps up was restarted: %d", got)
	}
	if Count(nil) != 0 {
		t.Fatal("none")
	}
}
