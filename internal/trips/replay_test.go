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
// "minutes ignition odometer [speed] [@lat,lon]"; the ignition "-" is an empty answer from Ford. The values are made up: only
// the shape of each real failure is kept, never a position, an odometer or a time from a real car.
const replayStart = "2026-10-22T16:00:00Z"

type replayer struct {
	t  *testing.T
	s  *store.Store
	d  *Detector
	t0 time.Time
}

func newReplayer(t *testing.T) *replayer {
	s, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	t0, _ := time.Parse(time.RFC3339, replayStart)
	return &replayer{t: t, s: s, d: &Detector{S: s, TZ: time.UTC}, t0: t0}
}

// feed runs more readings through the detector.
func (r *replayer) feed(fixture string) {
	r.t.Helper()
	for _, line := range strings.Split(strings.TrimSpace(fixture), "\n") {
		f := strings.Fields(line)
		if len(f) < 2 || strings.HasPrefix(f[0], "#") {
			continue
		}
		min, _ := strconv.ParseFloat(f[0], 64)
		ts := r.t0.Add(time.Duration(min * float64(time.Minute)))
		st := &ford.State{TS: ts}
		if f[1] != "-" {
			odo, _ := strconv.ParseFloat(f[2], 64)
			st = reading(ts, f[1], odo, 70)
			for _, tok := range f[3:] {
				if strings.HasPrefix(tok, "@") {
					ll := strings.Split(tok[1:], ",")
					la, _ := strconv.ParseFloat(ll[0], 64)
					lo, _ := strconv.ParseFloat(ll[1], 64)
					st.Lat, st.Lon = &la, &lo
				} else {
					sp, _ := strconv.ParseFloat(tok, 64)
					st.SpeedKph = &sp
				}
			}
			if _, err := r.s.SaveState("ford", st); err != nil { // the poller keeps every reading before the detector sees it
				r.t.Fatal(err)
			}
		}
		if err := r.d.Process(st); err != nil {
			r.t.Fatal(err)
		}
	}
}

// trips are the trips so far, newest first.
func (r *replayer) trips() []*store.Trip {
	ts, err := r.s.ListTrips(store.TripFilter{})
	if err != nil {
		r.t.Fatal(err)
	}
	return ts
}

func replay(t *testing.T, fixture string) []*store.Trip {
	t.Helper()
	r := newReplayer(t)
	r.feed(fixture)
	return r.trips()
}

func kms(ts []*store.Trip) []float64 {
	var out []float64
	for i := len(ts) - 1; i >= 0; i-- { // oldest first
		out = append(out, ts[i].DistanceKm)
	}
	return out
}
