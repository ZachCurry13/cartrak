package server

import (
	"math"

	"github.com/ZachCurry13/cartrak/internal/charging"
)

// A trip without a phone track has only the car's few positions, and joining them with straight lines cuts across fields.
// When the trip has an estimated road (the line its map shows), the replay follows that road instead: each timed car
// position is pinned to the nearest point of the road ahead of the previous pin, and the road between two pins is timed
// evenly by distance, so the car moves along the road at the pace it really went, stops included.

const pinWithinM = 2000 // a car position further than this from the road is not used to time it

func roadDist(a, b [2]float64) float64 {
	return charging.DistanceM(charging.Point{Lat: a[0], Lon: a[1]}, charging.Point{Lat: b[0], Lon: b[1]})
}

type pin struct {
	idx int
	ts  int64
}

// onRoad times the road from the car's positions; start and end are the trip's times.
func onRoad(road [][2]float64, car []replayPoint, start, end int64) []replayPoint {
	if len(road) < 2 || end <= start {
		return nil
	}
	along := make([]float64, len(road)) // meters from the start of the road
	for i := 1; i < len(road); i++ {
		along[i] = along[i-1] + roadDist(road[i-1], road[i])
	}
	pins := []pin{{0, start}}
	for _, c := range car {
		last := pins[len(pins)-1]
		if c.TS <= last.ts || c.TS >= end {
			continue
		}
		at := [2]float64{c.Lat, c.Lon}
		best, bestM := -1, 0.0
		for i := last.idx; i < len(road); i++ {
			if d := roadDist(at, road[i]); best < 0 || d < bestM-50 { // the earliest of near-equal spots, so a road that comes back is not skipped to
				best, bestM = i, d
			}
		}
		if best >= 0 && bestM <= pinWithinM {
			pins = append(pins, pin{best, c.TS})
		}
	}
	pins = append(pins, pin{len(road) - 1, end})
	var out []replayPoint
	for k := 0; k+1 < len(pins); k++ {
		a, b := pins[k], pins[k+1]
		span, secs := along[b.idx]-along[a.idx], float64(b.ts-a.ts)
		var kph *float64
		if secs > 0 && span > 0 {
			v := span / secs * 3.6
			kph = &v
		}
		if a.idx == b.idx { // stood still: the next point comes later
			out = append(out, replayPoint{TS: a.ts, Lat: road[a.idx][0], Lon: road[a.idx][1]})
			continue
		}
		for i := a.idx; i < b.idx; i++ {
			ts := a.ts
			if span > 0 {
				ts += int64(math.Round(secs * (along[i] - along[a.idx]) / span))
			}
			out = append(out, replayPoint{TS: ts, Lat: road[i][0], Lon: road[i][1], Kph: kph})
		}
	}
	lastPt := road[len(road)-1]
	return append(out, replayPoint{TS: end, Lat: lastPt[0], Lon: lastPt[1]})
}
