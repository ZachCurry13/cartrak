package mileage

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/geocode"
	"github.com/ZachCurry13/cartrak/internal/store"
)

// fakePlaces knows a few places by their rounded coordinates and counts the lookups it is asked for.
type fakePlaces struct {
	names   map[string]string
	cached  map[string]bool
	lookups int
	fail    bool
}

func (f *fakePlaces) Cached(lat, lon float64) (string, bool) {
	k := geocode.Key(lat, lon)
	if f.cached[k] {
		return f.names[k], true
	}
	return "", false
}

func (f *fakePlaces) Lookup(ctx context.Context, lat, lon float64) (string, error) {
	f.lookups++
	if f.fail {
		return "", errors.New("HTTP 429")
	}
	return f.names[geocode.Key(lat, lon)], nil
}

// Each city sits at its own spot so trips can point at it.
var spot = map[string][2]float64{
	"Springfield": {12.10, 23.10}, "Rivertown": {12.20, 23.20}, "Port Lakeside": {12.30, 23.30},
	"Oakdale": {12.40, 23.40}, "Fairview": {12.50, 23.50},
}

func places(cached bool) *fakePlaces {
	f := &fakePlaces{names: map[string]string{}, cached: map[string]bool{}}
	for name, p := range spot {
		k := geocode.Key(p[0], p[1])
		f.names[k], f.cached[k] = name, cached
	}
	return f
}

func trip(start time.Time, km float64, from, to string) *store.Trip {
	end := start.Add(20 * time.Minute).Unix()
	f, t := spot[from], spot[to]
	return &store.Trip{StartTS: start.Unix(), EndTS: &end, DistanceKm: km, Category: "work", TZ: "America/New_York",
		StartLat: &f[0], StartLon: &f[1], EndLat: &t[0], EndLon: &t[1]}
}

var local, _ = time.LoadLocation("America/New_York")

func at(day, hour int) time.Time { return time.Date(2026, 9, day, hour, 0, 0, 0, local) }

func TestADayListsEachParkedCityOnceAndLeavesOutHome(t *testing.T) {
	trips := []*store.Trip{
		trip(at(15, 8), 40, "Springfield", "Rivertown"),
		trip(at(15, 10), 20, "Rivertown", "Port Lakeside"),
		trip(at(15, 12), 30, "Port Lakeside", "Oakdale"),
		trip(at(15, 14), 10, "Oakdale", "Rivertown"),     // back to a city already listed
		trip(at(15, 16), 32, "Rivertown", "Springfield"), // home again: not a destination
		trip(at(16, 9), 25, "Springfield", "Oakdale"),
		trip(at(16, 15), 25, "Oakdale", "Springfield"),
	}
	rep := Build(context.Background(), trips, places(true), Options{Default: local})
	if len(rep.Rows) != 2 {
		t.Fatalf("rows: %+v", rep.Rows)
	}
	d1 := rep.Rows[0]
	if d1.Date != "2026-09-15" || d1.Origin != "Springfield" || d1.Destinations != "Rivertown, Port Lakeside, Oakdale" || d1.Description != "Customer Visits" {
		t.Fatalf("day one: %+v", d1)
	}
	if d1.Miles != 82 { // 132 km is 82.0 miles
		t.Fatalf("132 km should be 82 miles, got %d", d1.Miles)
	}
	if rep.Rows[1].Destinations != "Oakdale" || rep.Rows[1].Miles != 31 {
		t.Fatalf("day two: %+v", rep.Rows[1])
	}
	if rep.TotalMiles != 82+31 {
		t.Fatalf("the total adds the rows as shown: %d", rep.TotalMiles)
	}
}

func TestTheCopyBlockMatchesTheSheet(t *testing.T) {
	rep := Build(context.Background(), []*store.Trip{trip(at(15, 8), 100, "Springfield", "Rivertown"), trip(at(15, 12), 30, "Rivertown", "Springfield")},
		places(true), Options{Default: local, Description: "Customer Visits"})
	if got := rep.TSV(false); got != "2026-09-15\tSpringfield\tRivertown\tCustomer Visits\t81\n" {
		t.Fatalf("rows only: %q", got)
	}
	full := rep.TSV(true)
	if !strings.HasPrefix(full, "Date\tOrigin City\tDestination City\tDescription\tTotal Miles Driven\n") || !strings.HasSuffix(full, "\nTotal Mileage Recorded:\t\t\t\t81\n") {
		t.Fatalf("with header and total: %q", full)
	}
}

func TestOnlyFinishedTripsAreCountedAndDaysFollowTheTripsOwnClock(t *testing.T) {
	open := trip(at(15, 8), 10, "Springfield", "Rivertown")
	open.Open = true
	late := trip(time.Date(2026, 9, 15, 23, 30, 0, 0, local), 16.0934, "Springfield", "Oakdale") // 04:30 UTC the next day
	rep := Build(context.Background(), []*store.Trip{open, late}, places(true), Options{Default: time.UTC})
	if len(rep.Rows) != 1 || rep.Rows[0].Date != "2026-09-15" || rep.Rows[0].Miles != 10 || rep.Rows[0].Trips != 1 {
		t.Fatalf("an in-progress trip is skipped and a late-evening trip stays on its own day: %+v", rep.Rows)
	}
}

func TestLookupsAreLimitedAndReportedNotHidden(t *testing.T) {
	pl := places(false) // nothing cached yet
	trips := []*store.Trip{trip(at(15, 8), 40, "Springfield", "Rivertown"), trip(at(15, 10), 20, "Rivertown", "Oakdale"), trip(at(15, 12), 20, "Oakdale", "Springfield")}
	rep := Build(context.Background(), trips, pl, Options{Default: local, MaxLookups: 2})
	if pl.lookups != 2 || rep.Pending == 0 || len(rep.Notes) == 0 || !strings.Contains(rep.Notes[0], "Run the report again") {
		t.Fatalf("lookups %d, pending %d, notes %v", pl.lookups, rep.Pending, rep.Notes)
	}
	// A refusing service stops the run instead of hammering it.
	bad := places(false)
	bad.fail = true
	rep = Build(context.Background(), trips, bad, Options{Default: local})
	if bad.lookups != 1 || !strings.Contains(strings.Join(rep.Notes, " "), "HTTP 429") {
		t.Fatalf("one failed lookup should end the run: %d lookups, %v", bad.lookups, rep.Notes)
	}
}

func TestTripsWithoutAnEndLocationAreMentioned(t *testing.T) {
	tr := trip(at(15, 8), 40, "Springfield", "Rivertown")
	tr.EndLat, tr.EndLon = nil, nil
	rep := Build(context.Background(), []*store.Trip{tr}, places(true), Options{Default: local})
	if rep.Rows[0].Destinations != "" || !strings.Contains(strings.Join(rep.Notes, " "), "no end location") {
		t.Fatalf("%+v %v", rep.Rows[0], rep.Notes)
	}
}

func TestISOWeeks(t *testing.T) {
	loc := time.UTC
	s, e, err := WeekRange(2026, 26, 28, loc)
	if err != nil || s.Format("2006-01-02") != "2026-06-22" || e.Format("2006-01-02") != "2026-07-13" {
		t.Fatalf("weeks 26-28 of 2026 run Mon Jun 22 to Sun Jul 12: %v to %v (%v)", s, e, err)
	}
	if s, _, _ := WeekRange(2026, 1, 1, loc); s.Format("2006-01-02") != "2025-12-29" {
		t.Fatalf("ISO week 1 of 2026 starts on 2025-12-29, got %v", s)
	}
	if _, _, err := WeekRange(2025, 52, 53, loc); err == nil {
		t.Fatal("2025 has no week 53")
	}
	if _, _, err := WeekRange(2026, 53, 53, loc); err != nil {
		t.Fatalf("2026 has 53 weeks: %v", err)
	}
	for in, want := range map[string][2]int{"26": {26, 26}, "26-28": {26, 28}, "26 – 28": {26, 28}, "26 to 28": {26, 28}, " 3 - 4 ": {3, 4}} {
		if a, b, err := ParseWeeks(in); err != nil || a != want[0] || b != want[1] {
			t.Errorf("%q: %d %d %v", in, a, b, err)
		}
	}
	for _, bad := range []string{"", "x", "28-26", "0-3", "50-60", "1-2-3"} {
		if _, _, err := ParseWeeks(bad); err == nil {
			t.Errorf("%q should be refused", bad)
		}
	}
}

func TestEachTripCanBeRoundedUpToTheNextMile(t *testing.T) {
	// 0.16 km is 0.1 mi, 42.2 km is 26.2 mi, and 16.0934 km is exactly 10 mi.
	trips := []*store.Trip{
		trip(at(15, 8), 0.16, "Springfield", "Rivertown"),
		trip(at(15, 10), 42.2, "Rivertown", "Oakdale"),
		trip(at(15, 14), 16.09344, "Oakdale", "Springfield"),
	}
	up := Build(context.Background(), trips, places(true), Options{Default: local, RoundUp: true})
	if up.Rows[0].Miles != 1+27+10 || up.TotalMiles != 38 {
		t.Fatalf("0.1 becomes 1, 26.2 becomes 27 and a whole 10 stays 10: %+v", up.Rows[0])
	}
	near := Build(context.Background(), trips, places(true), Options{Default: local})
	if near.Rows[0].Miles != 36 { // 26.2 + 0.1 + 10 = 36.3
		t.Fatalf("the old way adds the day first: %+v", near.Rows[0])
	}
}
