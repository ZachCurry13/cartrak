package periodstats

import (
	"testing"
	"time"
)

func TestDayBarsCoverTheMonthAndChargingSplitsByPlaceKind(t *testing.T) {
	loc := time.UTC
	at := time.Date(2026, 10, 15, 12, 0, 0, 0, loc)
	from, to, ok := Range("day", at, 0, at)
	if !ok || !to.Equal(time.Date(2026, 11, 1, 0, 0, 0, 0, loc)) {
		t.Fatalf("range %v %v", from, to)
	}
	d := func(day, hour int) int64 { return time.Date(2026, 10, day, hour, 0, 0, 0, loc).Unix() }
	rep := Build("day", from, to,
		[]Trip{{Start: d(2, 8), DistanceKm: 10, Seconds: 600}, {Start: d(2, 17), DistanceKm: 5, Seconds: 300}, {Start: d(31, 23), DistanceKm: 1, Seconds: 60}},
		[]Charge{{Start: d(2, 20), AddedKwh: 10, Cost: 1.5, Seconds: 3600, Kind: "home"}, {Start: d(3, 12), AddedKwh: 30, Cost: 12, Kind: "public"}, {Start: d(4, 9), AddedKwh: 5, Kind: "work"}})
	if len(rep.Buckets) != 31 || rep.Buckets[1].Label != "2" || rep.Buckets[1].Km != 15 || rep.Buckets[1].DrivingSec != 900 || rep.Buckets[30].Km != 1 {
		t.Fatalf("bars: %+v", rep.Buckets[:3])
	}
	if rep.Buckets[1].Kwh != 10 || rep.Buckets[1].Cost != 1.5 {
		t.Fatalf("charging on the 2nd: %+v", rep.Buckets[1])
	}
	if len(rep.Kinds) != 3 || rep.Kinds[0].Kind != "home" || rep.Kinds[1].Kind != "work" || rep.Kinds[2].Kwh != 30 {
		t.Fatalf("kinds: %+v", rep.Kinds)
	}
}

func TestMonthAndYearBars(t *testing.T) {
	at := time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)
	from, to, _ := Range("month", at, 0, at)
	rep := Build("month", from, to, []Trip{{Start: time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC).Unix(), DistanceKm: 9}}, nil)
	if len(rep.Buckets) != 12 || rep.Buckets[6].Label != "Jul" || rep.Buckets[6].Km != 9 {
		t.Fatalf("%+v", rep.Buckets)
	}
	first := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC).Unix()
	from, to, _ = Range("year", at, first, at)
	rep = Build("year", from, to, nil, nil)
	if rep.Buckets[0].Label != "2022" || rep.Buckets[len(rep.Buckets)-1].Label != "2026" {
		t.Fatalf("years: %+v", rep.Buckets)
	}
	if _, _, ok := Range("week", at, 0, at); ok {
		t.Fatal("no week bars")
	}
}

func TestPlaceKinds(t *testing.T) {
	for _, c := range []struct{ where, place, want string }{{"home", "", "home"}, {"", "Work", "work"}, {"home", "Main Office", "work"}, {"public", "Mall", "public"}, {"", "", "public"}} {
		if got := KindOf(c.where, c.place); got != c.want {
			t.Fatalf("%v: %s", c, got)
		}
	}
}
