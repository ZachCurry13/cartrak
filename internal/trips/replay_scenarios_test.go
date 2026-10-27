package trips

import (
	"reflect"
	"testing"
)

// Each scenario is a real failure that once split, merged or lost a drive. The expected trips are the ones a person would call
// right: the distance of each drive, oldest first.
var replayScenarios = []struct {
	name    string
	fixture string
	want    []float64
	open    bool // a trip is still going at the end
}{
	{"an empty answer in the middle of a drive does not split it", `
0 OFF 1000
1 ON 1000 100
10 ON 1015 100
11 -
21 ON 1030 100
30 OFF 1040`, []float64{40}, false},
	{"a car that goes quiet for half an hour mid-drive is still one trip", `
0 OFF 1000
1 ON 1000 80
10 ON 1010 80
40 ON 1030 80
50 OFF 1035`, []float64{35}, false},
	{"two drives with the car parked between are two trips", `
0 OFF 1000
1 ON 1000 60
15 ON 1010 60
20 OFF 1012
200 OFF 1012
201 ON 1012 50
215 ON 1020 50
220 OFF 1022`, []float64{12, 10}, false},
	{"an old lower odometer on the last reading does not zero the drive", `
0 OFF 1000
1 ON 1000 80
10 ON 1019 80
20 OFF 1000`, []float64{19}, false},
	{"sitting in the car with the ignition on is not a trip", `
0 OFF 1000
1 ON 1000 0
15 ON 1000 0
30 ON 1000 0
31 OFF 1000`, nil, false},
	{"a drive first seen at its end, car on and stopped, still makes its trip", `
0 OFF 1000
30 ON 1012 0
35 OFF 1012`, []float64{12}, false},
	{"a car that only woke up a little further on is not a trip", `
0 OFF 1000
30 ON 1000.4 0
35 OFF 1000.4`, nil, false},
	{"an odometer jump too big for the time is not a trip", `
0 OFF 1000
2 ON 1500 0
5 OFF 1500`, nil, false},
	{"a brief ignition blip is not a trip", `
0 OFF 1000
1 ON 1000 0
2 OFF 1000`, nil, false},
}

func TestReplayedFailuresMakeTheTripsAPersonWouldExpect(t *testing.T) {
	for _, sc := range replayScenarios {
		t.Run(sc.name, func(t *testing.T) {
			got := replay(t, sc.fixture)
			open := len(got) > 0 && got[0].Open
			if g := kms(got); !reflect.DeepEqual(g, sc.want) && !(len(g) == 0 && len(sc.want) == 0) || open != sc.open {
				t.Fatalf("trips %v (open %v), want %v (open %v)", kms(got), open, sc.want, sc.open)
			}
		})
	}
}

func TestAReplayedDriveWithNoStartPositionBeginsWhereTheCarWasParked(t *testing.T) {
	trips := replay(t, `
0 OFF 1000 @45.5,-84.7
60 ON 1000 80
70 ON 1010 80
80 OFF 1015`)
	if len(trips) != 1 || trips[0].StartLat == nil || *trips[0].StartLat != 45.5 || *trips[0].StartLon != -84.7 {
		t.Fatalf("the drive began where it was parked: %+v", trips)
	}
}

func TestAReplayedStalePositionIsNotTheEndOfTheDriveUntilTheCarSaysWhere(t *testing.T) {
	r := newReplayer(t)
	r.feed(`
0 OFF 1000 @45.5,-84.7
1 ON 1000 80 @45.5,-84.7
10 ON 1019 80 @45.5,-84.7
20 OFF 1000 @45.5,-84.7`) // the car still reports where it started, and an old odometer
	trips := r.trips()
	if len(trips) != 1 || trips[0].DistanceKm != 19 || trips[0].EndLat != nil {
		t.Fatalf("19 km and the end unknown, not the start again: %+v", trips)
	}
	r.feed("25 OFF 1019 @45.9,-84.1")
	if got := r.trips(); got[0].EndLat == nil || *got[0].EndLat != 45.9 {
		t.Fatalf("the first new position becomes the end: %+v", got[0].EndLat)
	}
}
