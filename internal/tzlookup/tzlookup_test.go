package tzlookup

import (
	"testing"
	"time"
)

func TestKnownPlaces(t *testing.T) {
	cases := []struct {
		name     string
		lat, lon float64
		want     string
	}{
		{"Minneapolis", 44.9778, -93.2650, "America/Chicago"},
		{"New York City", 40.7128, -74.0060, "America/New_York"},
		{"Los Angeles", 34.0522, -118.2437, "America/Los_Angeles"},
		{"Denver", 39.7392, -104.9903, "America/Denver"},
		{"Phoenix (no daylight saving)", 33.4484, -112.0740, "America/Phoenix"},
		{"Detroit", 42.3314, -83.0458, "America/Detroit"},
		{"Toronto", 43.6532, -79.3832, "America/Toronto"},
		{"London", 51.5074, -0.1278, "Europe/London"},
		{"Tokyo", 35.6762, 139.6503, "Asia/Tokyo"},
	}
	for _, c := range cases {
		if got := Zone(c.lat, c.lon); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

func TestBadInputAndFallback(t *testing.T) {
	if Zone(95, 10) != "" || Zone(10, 200) != "" {
		t.Fatal("out-of-range coordinates should give no zone")
	}
	fb := time.FixedZone("fallback", 3600)
	if Location(999, 999, fb) != fb {
		t.Fatal("should fall back")
	}
	if l := Location(40.7128, -74.0060, fb); l.String() != "America/New_York" {
		t.Fatalf("location: %v", l)
	}
}

func TestSameInstantDiffersByPlace(t *testing.T) {
	at := time.Date(2026, 10, 14, 13, 30, 0, 0, time.UTC) // Wednesday
	mn := Location(44.9778, -93.2650, time.UTC)
	ny := Location(40.7128, -74.0060, time.UTC)
	if at.In(mn).Hour() != 8 || at.In(ny).Hour() != 9 {
		t.Fatalf("Minnesota %d:00, New York %d:00", at.In(mn).Hour(), at.In(ny).Hour())
	}
}
