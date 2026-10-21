package store

import (
	"context"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
)

func state(at time.Time, soc float64, mod func(*ford.State)) *ford.State {
	rng, odo, out := soc*3, 1000+soc, 15.0
	st := &ford.State{TS: at, SoCPct: &soc, RangeKm: &rng, OdometerKm: &odo, OutsideC: &out, Ignition: "OFF", Raw: []byte(`{}`),
		Extra: map[string]any{"telemetry.metrics.x.value": 1}}
	if mod != nil {
		mod(st)
	}
	return st
}

func TestTheCarCanBeLookedUpAsItWasAtAnyMoment(t *testing.T) {
	s := openTemp(t)
	base := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	for i, soc := range []float64{80, 70, 60} {
		if _, err := s.SaveState("t", state(base.Add(time.Duration(i)*time.Hour), soc, nil)); err != nil {
			t.Fatal(err)
		}
	}
	at := func(min int) *float64 {
		x, err := s.SampleAt(base.Add(time.Duration(min) * time.Minute).Unix())
		if err != nil {
			t.Fatal(err)
		}
		if x == nil {
			return nil
		}
		return x.SoCPct
	}
	if got := at(-5); got != nil {
		t.Fatalf("before the first reading there is nothing: %v", *got)
	}
	if got := at(0); got == nil || *got != 80 {
		t.Fatalf("at the first reading: %v", got)
	}
	if got := at(90); got == nil || *got != 70 {
		t.Fatalf("between readings the earlier one applies (it is what the car last said): %v", got)
	}
	if got := at(5000); got == nil || *got != 60 {
		t.Fatalf("after the last reading: %v", got)
	}
	prev, next := s.NeighborTS(base.Add(time.Hour).Unix())
	if prev != base.Unix() || next != base.Add(2*time.Hour).Unix() {
		t.Fatalf("neighbors: %d %d", prev, next)
	}
	if p, n := s.NeighborTS(base.Unix()); p != 0 || n == 0 {
		t.Fatalf("no reading before the first: %d %d", p, n)
	}
	if s.FirstSampleTS() != base.Unix() {
		t.Fatal("the first reading time is for the date picker")
	}
}

func TestTimelineComesFromTheSmallTableAndIsThinned(t *testing.T) {
	s := openTemp(t)
	base := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 100; i++ {
		driving := i >= 40 && i < 60
		s.SaveState("t", state(base.Add(time.Duration(i)*10*time.Minute), 90-float64(i)/2, func(st *ford.State) {
			if driving {
				st.Ignition = "ON"
			}
		}))
	}
	all, err := s.Timeline(base.Unix(), base.Add(48*time.Hour).Unix(), 0)
	if err != nil || len(all) != 100 {
		t.Fatalf("all points: %d %v", len(all), err)
	}
	if !all[45].Driving || all[10].Driving || all[45].SoC == nil || all[45].Odo == nil || all[45].OutsideC == nil {
		t.Fatalf("each point carries what the car was doing and a few numbers: %+v", all[45])
	}
	thin, _ := s.Timeline(base.Unix(), base.Add(48*time.Hour).Unix(), 25)
	if len(thin) != 25 || thin[0].TS != all[0].TS || thin[24].TS != all[99].TS {
		t.Fatalf("thinned to 25 and keeping both ends: %d", len(thin))
	}
	if part, _ := s.Timeline(base.Add(500*time.Minute).Unix(), base.Add(600*time.Minute).Unix(), 0); len(part) != 11 {
		t.Fatalf("the window limits the points: %d", len(part))
	}
}

func TestOlderReadingsGainTheirTimelineValuesFromTheBackfill(t *testing.T) {
	s := openTemp(t)
	base := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 12; i++ {
		s.SaveState("t", state(base.Add(time.Duration(i)*time.Minute), 80, func(st *ford.State) { st.Ignition = "ON" }))
	}
	// Pretend they were saved before the table had its timeline columns.
	s.DB.Exec(`UPDATE trends SET range_km=NULL, odometer_km=NULL, driving=0`)
	s.SetKV(kvBackfill, "")
	if err := s.BackfillTrends(context.Background(), 5, time.Millisecond); err != nil {
		t.Fatal(err)
	}
	tl, _ := s.Timeline(base.Unix(), base.Add(time.Hour).Unix(), 0)
	if len(tl) != 12 || !tl[3].Driving || tl[3].RangeKm == nil || tl[3].Odo == nil {
		t.Fatalf("the backfill must restore range, odometer and the driving flag: %+v", tl[3])
	}
}
