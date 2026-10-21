package server

import "testing"

func TestReplayFollowsTheRoadAtTheCarsPace(t *testing.T) {
	// an L-shaped road: east, then north; the car reported once, at the corner, a third of the way through the trip's time
	road := [][2]float64{{45, -85}, {45, -84.99}, {45, -84.98}, {45.01, -84.98}, {45.02, -84.98}}
	car := []replayPoint{{TS: 1100, Lat: 45.0003, Lon: -84.9801}}
	pts := onRoad(road, car, 1000, 1300)
	if len(pts) != len(road) {
		t.Fatalf("one point per road point: %d", len(pts))
	}
	for i, p := range pts {
		if p.Lat != road[i][0] || p.Lon != road[i][1] {
			t.Fatalf("point %d is off the road: %+v", i, p)
		}
		if i > 0 && p.TS < pts[i-1].TS {
			t.Fatalf("time goes back at %d", i)
		}
	}
	if pts[0].TS != 1000 || pts[2].TS != 1100 || pts[4].TS != 1300 {
		t.Fatalf("pinned to the car's corner time and the trip's ends: %d %d %d", pts[0].TS, pts[2].TS, pts[4].TS)
	}
	if pts[1].TS != 1050 || pts[3].TS != 1200 {
		t.Fatalf("timed evenly by distance between pins: %d %d", pts[1].TS, pts[3].TS)
	}
	if pts[0].Kph == nil || pts[2].Kph == nil || *pts[0].Kph <= *pts[2].Kph {
		t.Fatal("faster on the first leg (same distance in half the time)")
	}
	far := []replayPoint{{TS: 1100, Lat: 46, Lon: -84}}          // nowhere near the road: not used
	if pts := onRoad(road, far, 1000, 1300); pts[2].TS != 1124 { // timed by distance alone: the corner is 1.58 of the road's 3.8 km
		t.Fatalf("a far position is ignored: %d", pts[2].TS)
	}
}
