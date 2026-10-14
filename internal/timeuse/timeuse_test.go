package timeuse

import (
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

func f(v float64) *float64 { return &v }

func trip(from, to int64, endLat float64) *store.Trip {
	return &store.Trip{StartTS: from, EndTS: &to, EndLat: f(endLat), EndLon: f(-75)}
}

func TestADaySplitsIntoDrivingChargingAndParkedWithNoDoubleCounting(t *testing.T) {
	day := time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC).Unix()
	h := int64(3600)
	ts := []*store.Trip{trip(day+8*h, day+9*h, 40.05), trip(day+17*h, day+18*h, 40.0)} // office, then home
	end := day + 22*h
	cs := []*store.Charge{{StartTS: day + 18*h + 30*60, EndTS: &end}, {StartTS: day + 8*h + 30*60, EndTS: func() *int64 { v := day + 9*h + 30*60; return &v }()}} // the second overlaps a drive
	named := []*store.NamedPlace{{Name: "Home", Lat: 40.0, Lon: -75, RadiusM: 150}, {Name: "Office", Lat: 40.05, Lon: -75, RadiusM: 150}}
	rep := Analyze(ts, cs, named, day, day+24*h, 0, time.UTC)
	if rep.Driving != 2*h {
		t.Fatalf("driving: %d", rep.Driving)
	}
	// Charging 18:30-22:00 = 3.5 h, plus 09:00-09:30 (the part of the other session after the drive ended) = 0.5 h.
	if rep.Charging != 4*h || rep.Parked != 24*h-2*h-4*h {
		t.Fatalf("charging %d parked %d", rep.Charging, rep.Parked)
	}
	if len(rep.Days) != 1 || rep.Days[0].Driving != 2*h || rep.Days[0].Parked != 18*h {
		t.Fatalf("days: %+v", rep.Days)
	}
	got := map[string]int64{}
	for _, p := range rep.Places {
		got[p.Name] = p.Sec
	}
	// Office: 09:00 to 17:00 less 0.5 h of charging = 7.5 h. Home: 18:00 to 24:00 less 3.5 h charging = 2.5 h.
	if got["Office"] != 7*h+30*60 || got["Home"] != 2*h+30*60 {
		t.Fatalf("places: %+v", rep.Places)
	}
}

func TestTimeBeforeTheFirstReadingIsNotCountedAndLastPlaceIsElsewhereWhenUnnamed(t *testing.T) {
	h := int64(3600)
	rep := Analyze([]*store.Trip{trip(10*h, 11*h, 41)}, nil, nil, 0, 20*h, 5*h, time.UTC)
	if rep.From != 5*h || rep.Driving != h || rep.Parked != 20*h-5*h-h {
		t.Fatalf("%+v", rep)
	}
	if len(rep.Places) != 1 || rep.Places[0].Name != "Elsewhere" || rep.Places[0].Sec != 9*h {
		t.Fatalf("%+v", rep.Places)
	}
	if empty := Analyze(nil, nil, nil, 100, 100, 0, time.UTC); empty.Parked != 0 || len(empty.Days) != 0 {
		t.Fatal("nothing in an empty window")
	}
}
