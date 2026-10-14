package datacheck

import "testing"

func kinds(fs []Finding) map[string]int {
	m := map[string]int{}
	for _, f := range fs {
		m[f.Kind]++
	}
	return m
}

func TestOdometerGapsAndGoingBackwards(t *testing.T) {
	d := int64(86400)
	got := kinds(Check([]Odo{{0, 100}, {d / 2, 101}, {d/2 + 3*d, 102}, {d/2 + 3*d + 100, 90}}, nil))
	if got["gap"] != 1 || got["odo-back"] != 1 {
		t.Fatalf("%v", got)
	}
	if len(Check([]Odo{{0, 100}, {3600, 100.2}, {7200, 100.1}}, nil)) != 0 {
		t.Fatal("small wobble is not a finding")
	}
}

func TestTripsThatCannotBeRight(t *testing.T) {
	h := int64(3600)
	trips := []Trip{
		{ID: 1, StartTS: 0, EndTS: h, DistanceKm: 30},
		{ID: 2, StartTS: h / 2, EndTS: 2 * h, DistanceKm: 20},         // overlaps 1
		{ID: 3, StartTS: 10 * h, EndTS: 10*h + 600, DistanceKm: 0.05}, // tiny
		{ID: 4, StartTS: 20 * h, EndTS: 20*h + 600, DistanceKm: 80},   // 480 km/h
		{ID: 5, StartTS: 30 * h, EndTS: 70 * h, DistanceKm: 10},       // 40 hours
		{ID: 6, StartTS: 100 * h, Open: true},                         // still going: never judged
	}
	got := Check(nil, trips)
	m := map[int64]string{}
	for _, f := range got {
		m[f.TripID] = f.Kind
	}
	if m[2] != "overlap" || m[3] != "tiny" || m[4] != "fast" || m[5] != "long" || m[1] != "" || m[6] != "" {
		t.Fatalf("%+v", got)
	}
	if got[0].TripID != 5 {
		t.Fatalf("newest first: %+v", got[0])
	}
}
