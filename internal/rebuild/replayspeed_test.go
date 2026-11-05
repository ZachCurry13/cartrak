package rebuild

import (
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
	"github.com/ZachCurry13/cartrak/internal/trips"
)

// A month of readings: a parked reading every ten minutes and a half-hour drive each day with a reading every 30 seconds.
func monthOfRows() []store.ReplayRow {
	var rows []store.ReplayRow
	t0 := time.Now().Add(-40 * 24 * time.Hour).Truncate(time.Hour).Unix()
	odo := 1000.0
	for day := 0; day < 30; day++ {
		base := t0 + int64(day)*86400
		for m := 0; m < 24*6; m++ { // parked, every 10 minutes
			ts := base + int64(m)*600
			if m >= 48 && m < 51 { // the drive: 08:00 to 08:30
				continue
			}
			rows = append(rows, store.ReplayRow{TS: ts, OdometerKm: f(odo), SoC: f(80), Lat: f(44), Lon: f(-93), SpeedKph: f(0), Ignition: "OFF"})
		}
		for k := 0; k < 60; k++ { // driving
			odo += 0.7
			rows = append(rows, store.ReplayRow{TS: base + 8*3600 + int64(k)*30, OdometerKm: f(odo), SoC: f(80 - float64(k)/20), Lat: f(44 + float64(k)/1000), Lon: f(-93), SpeedKph: f(80), Ignition: "ON"})
		}
	}
	return rows
}

func TestReplayingAMonthOfReadingsIsQuick(t *testing.T) {
	rows := monthOfRows()
	start := time.Now()
	nt, _, err := replay(rows, &trips.Detector{TZ: time.UTC})
	took := time.Since(start)
	t.Logf("%d readings replayed in %v into %d trips", len(rows), took, len(nt))
	if err != nil || len(nt) < 25 {
		t.Fatalf("%v, %d trips", err, len(nt))
	}
	limit := 8 * time.Second // a month replays in about a second; the race detector makes it about twenty times slower
	if raceOn {
		limit = 120 * time.Second
	}
	if took > limit {
		t.Fatalf("a month took %v: the scratch database is syncing every write", took)
	}
}
