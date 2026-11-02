package server

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

func TestTheSummaryCarriesDrivingTimePerCategorySoSharesCanBeShown(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	lat, lon := 40.0, -75.0
	add := func(category string, km float64, minutes int, hoursAgo int) {
		start := time.Now().Add(-time.Duration(hoursAgo) * time.Hour)
		end := start.Add(time.Duration(minutes) * time.Minute).Unix()
		o1, o2 := 1000.0+float64(hoursAgo), 1000.0+float64(hoursAgo)+km
		tr := &store.Trip{StartTS: start.Unix(), EndTS: &end, StartLat: &lat, StartLon: &lon, EndLat: &lat, EndLon: &lon, StartOdo: &o1, EndOdo: &o2, DistanceKm: km, Category: category, TZ: "UTC"}
		if err := srv.S.AddManualTrip(tr); err != nil {
			t.Fatal(err)
		}
	}
	add("work", 30, 30, 5)
	add("work", 10, 10, 4)
	add("personal", 20, 60, 3)
	_, body := do(t, admin, "GET", ts.URL+"/api/summary", "", nil)
	var out map[string]struct {
		Trips      int
		DistanceKm float64
		Seconds    float64
	}
	if json.Unmarshal([]byte(body), &out) != nil {
		t.Fatalf("%s", body)
	}
	if w := out["work"]; w.Trips != 2 || w.DistanceKm != 40 || w.Seconds != 40*60 {
		t.Fatalf("work: %+v", w)
	}
	if p := out["personal"]; p.Trips != 1 || p.Seconds != 3600 {
		t.Fatalf("personal: %+v", p)
	}
}
