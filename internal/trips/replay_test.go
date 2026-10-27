package trips

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/store"
)

// A replay fixture is a few lines of readings the way Ford sent them, fed through the detector one by one. Every line is
// "minutes ignition odometer [speed]"; the ignition "-" is an empty answer from Ford. The values are made up: only the shape of
// each real failure is kept, never a position, an odometer or a time from a real car.
const replayStart = "2026-10-22T16:00:00Z"

func replay(t *testing.T, fixture string) []*store.Trip {
	t.Helper()
	s, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	d := &Detector{S: s, TZ: time.UTC}
	t0, _ := time.Parse(time.RFC3339, replayStart)
	for _, line := range strings.Split(strings.TrimSpace(fixture), "\n") {
		f := strings.Fields(line)
		if len(f) < 2 || strings.HasPrefix(f[0], "#") {
			continue
		}
		min, _ := strconv.ParseFloat(f[0], 64)
		ts := t0.Add(time.Duration(min * float64(time.Minute)))
		var st *ford.State
		if f[1] == "-" {
			st = &ford.State{TS: ts}
		} else {
			odo, _ := strconv.ParseFloat(f[2], 64)
			st = reading(ts, f[1], odo, 70)
			if len(f) > 3 {
				sp, _ := strconv.ParseFloat(f[3], 64)
				st.SpeedKph = &sp
			}
		}
		if err := d.Process(st); err != nil {
			t.Fatal(err)
		}
	}
	trips, err := s.ListTrips(store.TripFilter{})
	if err != nil {
		t.Fatal(err)
	}
	return trips
}

func kms(ts []*store.Trip) []float64 {
	var out []float64
	for i := len(ts) - 1; i >= 0; i-- { // oldest first
		out = append(out, ts[i].DistanceKm)
	}
	return out
}
