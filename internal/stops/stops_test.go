package stops

import "testing"

func f(v float64) *float64 { return &v }

const m = 60

func TestAStopIsTheGapAfterATripTakingThatTripsCategory(t *testing.T) {
	trips := []Trip{
		{ID: 1, StartTS: 0, EndTS: 1000, Category: "work", EndLat: f(1), EndLon: f(2)},
		{ID: 2, StartTS: 1000 + 45*m, EndTS: 1000 + 60*m, Category: "personal"},
		{ID: 3, StartTS: 1000 + 60*m + 20*m, EndTS: 1000 + 80*m, Category: "work"},
	}
	got := Build(trips, nil, nil, 1000+90*m)
	if len(got) != 3 || got[0].TripID != 1 || got[0].Minutes != 45 || got[0].Category != "work" || got[0].Open {
		t.Fatalf("a 45 minute stop after the work trip: %+v", got)
	}
	if got[1].TripID != 2 || got[1].Minutes != 20 || got[1].Category != "personal" {
		t.Fatalf("the next stop: %+v", got[1])
	}
	if !got[2].Open || got[2].Minutes != 10 {
		t.Fatalf("the car is still parked after the last trip: %+v", got[2])
	}
}

func TestShortAndVeryLongGapsAreNotStops(t *testing.T) {
	trips := []Trip{
		{ID: 1, StartTS: 0, EndTS: 100, Category: "work"},
		{ID: 2, StartTS: 100 + 9*m, EndTS: 100 + 20*m, Category: "work"},                          // 9 minutes: a red light
		{ID: 3, StartTS: 100 + 20*m + 9*60*m, EndTS: 100 + 20*m + 9*60*m + 600, Category: "work"}, // 9 hours: parked overnight
	}
	if got := Build(trips, nil, nil, 100+20*m+9*60*m+601); len(got) != 0 {
		t.Fatalf("no stops: %+v", got)
	}
}

func TestAPersonsLabelWinsAndANoteAloneKeepsTheCategory(t *testing.T) {
	trips := []Trip{{ID: 1, StartTS: 0, EndTS: 100, Category: "work"}, {ID: 2, StartTS: 100 + 30*m, EndTS: 100 + 40*m, Category: "work"}}
	got := Build(trips, map[int64]Label{1: {Category: "personal", Note: "lunch"}}, nil, 100+40*m)
	if got[0].Category != "personal" || !got[0].Manual || got[0].Note != "lunch" {
		t.Fatalf("manual label: %+v", got[0])
	}
	got = Build(trips, map[int64]Label{1: {Note: "Acme Corp"}}, nil, 100+40*m)
	if got[0].Category != "work" || got[0].Manual || got[0].Note != "Acme Corp" {
		t.Fatalf("a note does not change the category: %+v", got[0])
	}
}

func TestStopsAtHomeOrDuringAChargeAreLeftOut(t *testing.T) {
	trips := []Trip{
		{ID: 1, StartTS: 0, EndTS: 100, Category: "work", EndLat: f(1), EndLon: f(1)},
		{ID: 2, StartTS: 100 + 30*m, EndTS: 100 + 40*m, Category: "work", EndLat: f(9), EndLon: f(9)},
		{ID: 3, StartTS: 100 + 80*m, EndTS: 100 + 90*m, Category: "work"},
	}
	skip := func(lat, lon *float64, start, end int64) bool { return lat != nil && *lat == 1 } // home
	got := Build(trips, nil, skip, 100+90*m)
	if len(got) != 1 || got[0].TripID != 2 {
		t.Fatalf("the stop at home is left out, the other stays: %+v", got)
	}
}

func TestTotalsAddUpByDayAndWorkSeparately(t *testing.T) {
	ss := []Stop{{Start: 1, Minutes: 30, Category: "work"}, {Start: 2, Minutes: 20, Category: "personal"}, {Start: 100000, Minutes: 15, Category: "work"}}
	day := func(ts int64) string {
		if ts < 50000 {
			return "2026-10-26"
		}
		return "2026-10-27"
	}
	got := Totals(ss, day)
	if len(got) != 2 || got[0].Date != "2026-10-27" || got[0].Work != 15 || got[1].Minutes != 50 || got[1].Work != 30 {
		t.Fatalf("totals: %+v", got)
	}
}

func TestAWaitWithTheEngineOnAtTheEndOfATripBelongsToTheStop(t *testing.T) {
	// the car stopped moving at 1000 and the engine was switched off 13 minutes later; the next trip starts 20 minutes after that
	trips := []Trip{
		{ID: 1, StartTS: 0, EndTS: 1000 + 13*m, LastMoving: 1000, Category: "work"},
		{ID: 2, StartTS: 1000 + 33*m, EndTS: 1000 + 40*m, Category: "work"},
	}
	got := Build(trips, nil, nil, 1000+41*m)
	if len(got) < 1 || got[0].Start != 1000 || got[0].Minutes != 33 {
		t.Fatalf("the stop starts when the car stopped moving and lasts 33 minutes: %+v", got)
	}
	// a wait of under two minutes is just parking
	trips[0].LastMoving = trips[0].EndTS - 60
	if got := Build(trips, nil, nil, 1000+41*m); got[0].Start != trips[0].EndTS || got[0].Minutes != 20 {
		t.Fatalf("a short wait changes nothing: %+v", got[0])
	}
}
