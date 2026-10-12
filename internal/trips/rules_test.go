package trips

import (
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

func f(v float64) *float64 { return &v }

func TestClassifyWeekdayWindow(t *testing.T) {
	rules := []store.Rule{{ID: 1, Category: Work, Enabled: true, Days: 62, StartMin: 480, EndMin: 1020}}
	mon9 := time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC) // Monday
	mon18 := time.Date(2026, 10, 12, 18, 0, 0, 0, time.UTC)
	sat9 := time.Date(2026, 10, 17, 9, 0, 0, 0, time.UTC)
	if c, id, ok := Classify(rules, mon9, nil, nil); !ok || c != Work || id != 1 {
		t.Fatalf("mon 9am: %v %v %v", c, id, ok)
	}
	if _, _, ok := Classify(rules, mon18, nil, nil); ok {
		t.Fatal("5pm boundary is exclusive; 6pm must not match")
	}
	if _, _, ok := Classify(rules, sat9, nil, nil); ok {
		t.Fatal("saturday must not match")
	}
}

func TestClassifyPriorityAndGeo(t *testing.T) {
	rules := []store.Rule{
		{ID: 1, Category: Work, Enabled: true, Priority: 0, Days: 127, StartMin: 0, EndMin: 1440},
		{ID: 2, Category: Personal, Enabled: true, Priority: 10, Days: 127, StartMin: 0, EndMin: 1440,
			GeoLat: f(40), GeoLon: f(-80), GeoRadius: f(500)},
		{ID: 3, Category: Personal, Enabled: false, Priority: 99, Days: 127, StartMin: 0, EndMin: 1440},
	}
	at := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	if c, id, _ := Classify(rules, at, f(40.001), f(-80)); c != Personal || id != 2 {
		t.Fatalf("near geofence should hit rule 2, got %v %v", c, id)
	}
	if c, id, _ := Classify(rules, at, f(41), f(-80)); c != Work || id != 1 {
		t.Fatalf("far away falls to rule 1, got %v %v", c, id)
	}
	if c, _, _ := Classify(rules, at, nil, nil); c != Work {
		t.Fatalf("no gps: geofence rule skipped, got %v", c)
	}
}

func TestWindowWrapsMidnight(t *testing.T) {
	if !inWindow(23*60, 22*60, 6*60) || !inWindow(2*60, 22*60, 6*60) || inWindow(12*60, 22*60, 6*60) {
		t.Fatal("overnight window wrong")
	}
}
