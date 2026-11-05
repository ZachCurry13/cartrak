package analytics

import (
	"testing"
	"time"
)

func TestTheNewRecordsFindTheLongestPlugBusiestDayAndFarthestBetweenCharges(t *testing.T) {
	loc := time.UTC
	day := func(d, h int) int64 { return time.Date(2026, 10, d, h, 0, 0, 0, loc).Unix() }
	trips := []Trip{
		{ID: 1, Start: day(1, 9), Km: 40}, {ID: 2, Start: day(1, 12), Km: 30}, {ID: 3, Start: day(1, 15), Km: 20}, // three trips on the 1st
		{ID: 4, Start: day(3, 9), Km: 120}, {ID: 5, Start: day(3, 13), Km: 90}, // 210 km between the charges on the 1st and the 6th
		{ID: 6, Start: day(7, 9), Km: 50},
	}
	charges := []Charge{
		{ID: 10, Start: day(1, 18), End: day(1, 19), Kwh: 20},
		{ID: 11, Start: day(6, 8), End: day(6, 12), Kwh: 40}, // four hours: the longest
		{ID: 12, Start: day(8, 8), End: day(8, 9), Kwh: 10},
	}
	got := map[string]Record{}
	for _, r := range Records(trips, charges, loc) {
		got[r.Key] = r
	}
	if r := got["longestPlug"]; r.Value != 4 || r.ChargeID != 11 || r.Kind != "hours" {
		t.Errorf("longest plug: %+v", r)
	}
	if r := got["mostTrips"]; r.Value != 3 || r.Date != "2026-10-01" {
		t.Errorf("most trips: %+v", r)
	}
	if r := got["farthestCharge"]; r.Value != 210 || r.ChargeID != 11 {
		t.Errorf("farthest between charges: %+v", r)
	}
	if _, ok := Records(trips, charges[:2], loc)[0], true; ok {
		for _, r := range Records(trips, charges[:2], loc) {
			if r.Key == "farthestCharge" {
				t.Errorf("two charges are not enough to say what is farthest between them")
			}
		}
	}
}
