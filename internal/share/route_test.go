package share

import "testing"

// A straight line due north, one point every 100 m (0.0009 degrees of latitude is about 100 m).
func line(n int) [][2]float64 {
	out := make([][2]float64, n)
	for i := range out {
		out[i] = [2]float64{40 + float64(i)*0.0009, -75}
	}
	return out
}

func TestTrimLeavesOutTheEndsOfARoute(t *testing.T) {
	got := Trim(line(41), TrimMeters) // 4 km long
	if len(got) < 30 || len(got) > 35 {
		t.Fatalf("about 3.2 km should remain of 4: %d points", len(got))
	}
	first, last := got[0], got[len(got)-1]
	if meters(line(1)[0], first) < TrimMeters || meters([2]float64{40.036, -75}, last) < TrimMeters-1 {
		t.Fatalf("neither end may be closer than 400 m to the real start or finish: %v %v", first, last)
	}
	if Trim(line(8), TrimMeters) != nil {
		t.Fatal("a 700 m route has nothing left to show")
	}
}

func TestThinKeepsTheEndsAndRounds(t *testing.T) {
	got := Thin(line(1000), 50)
	if len(got) != 50 || got[0] != [2]float64{40, -75} || got[49][0] < 40.89 {
		t.Fatalf("%d points, first %v last %v", len(got), got[0], got[len(got)-1])
	}
	if p := Thin([][2]float64{{40.123456789, -75.987654321}, {41, -75}}, 10); p[0] != [2]float64{40.12346, -75.98765} {
		t.Fatalf("rounded to a metre: %v", p[0])
	}
}

func TestATileMustBeNearTheSharedRoute(t *testing.T) {
	bb := BBox(line(41))
	if bb != "40.00000,-75.00000,40.03600,-75.00000" {
		t.Fatalf("%q", bb)
	}
	// Tile coordinates of the route's area at zoom 12 (about 10 km wide tiles): x=1197 (lon -75) y=1553 (lat 40).
	if !TileAllowed(bb, 12, 1197, 1553) {
		t.Fatal("the tile over the route is allowed")
	}
	if TileAllowed(bb, 12, 1297, 1553) || TileAllowed(bb, 12, 1197, 1353) {
		t.Fatal("tiles far away are not")
	}
	if !TileAllowed(bb, 5, 1, 1) || TileAllowed(bb, 17, 1197, 1553) || TileAllowed("", 12, 1197, 1553) {
		t.Fatal("country views are fine; street level and unknown routes are not")
	}
}
