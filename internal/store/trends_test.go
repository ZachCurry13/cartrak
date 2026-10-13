package store

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
)

func reading(at time.Time, v12 float64) *ford.State {
	cap := 88.0
	lat, lon := 10.0, 20.0
	return &ford.State{TS: at, Battery12V: &v12, CapacityKwh: &cap, Lat: &lat, Lon: &lon, Raw: []byte(`{}`),
		Extra: map[string]any{"telemetry.metrics.big.value": strings.Repeat("x", 1000)}}
}

func TestChartedValuesAreSavedBesideTheReadingAndQueriedWithoutTheBigRows(t *testing.T) {
	s := openTemp(t)
	day := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 3; i++ {
		if _, err := s.SaveState("t", reading(day.Add(time.Duration(i)*time.Minute), 12.0+float64(i)/10)); err != nil {
			t.Fatal(err)
		}
	}
	var n int
	s.DB.QueryRow(`SELECT COUNT(*) FROM trends`).Scan(&n)
	if n != 3 {
		t.Fatalf("each reading should add one trends row, got %d", n)
	}
	days, err := s.DailySeries("capacityKwh", 0)
	if err != nil || len(days) != 1 || days[0].N != 3 || days[0].Avg != 88 {
		t.Fatalf("series: %+v %v", days, err)
	}
	// The point of the table: a chart must not touch the large sample rows at all.
	rows, err := s.DB.Query(`EXPLAIN QUERY PLAN SELECT AVG(capacity_kwh) FROM trends WHERE ts >= 0 AND capacity_kwh IS NOT NULL GROUP BY (ts/86400)`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var a, b, c int
		var detail string
		rows.Scan(&a, &b, &c, &detail)
		if strings.Contains(detail, "samples") {
			t.Fatalf("a chart query must not read the samples table: %s", detail)
		}
	}
}

func TestOlderReadingsAreCopiedInBatchesAndItResumes(t *testing.T) {
	s := openTemp(t)
	day := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 25; i++ {
		s.SaveState("t", reading(day.Add(time.Duration(i)*time.Minute), 12.5))
	}
	// Pretend these readings were saved by a version that had no trends table.
	s.DB.Exec(`DELETE FROM trends`)
	s.SetKV(kvBackfill, "")

	ctx, cancel := context.WithCancel(context.Background())
	if err := s.BackfillTrends(ctx, 10, time.Millisecond); err != nil {
		t.Fatal(err)
	}
	cancel()
	var n int
	s.DB.QueryRow(`SELECT COUNT(*) FROM trends`).Scan(&n)
	if n != 25 {
		t.Fatalf("all 25 older readings should be copied in batches of 10: %d", n)
	}
	if got := s.GetKV(kvBackfill); got == "" || got == "0" {
		t.Fatalf("progress must be remembered: %q", got)
	}
	// Running it again changes nothing, and a new reading is picked up from where it stopped.
	s.SaveState("t", reading(day.Add(time.Hour), 12.5))
	if err := s.BackfillTrends(context.Background(), 10, time.Millisecond); err != nil {
		t.Fatal(err)
	}
	s.DB.QueryRow(`SELECT COUNT(*) FROM trends`).Scan(&n)
	if n != 26 {
		t.Fatalf("no duplicates: %d", n)
	}
	// A run that is stopped part-way picks up where it left off.
	s.DB.Exec(`DELETE FROM trends`)
	s.SetKV(kvBackfill, "20")
	s.BackfillTrends(context.Background(), 10, time.Millisecond)
	s.DB.QueryRow(`SELECT COUNT(*) FROM trends`).Scan(&n)
	if n != 6 { // ids 21 to 26
		t.Fatalf("resume after id 20 should copy the remaining 6, got %d", n)
	}
}

func TestTrackBetweenIsLightButKeepsThePositions(t *testing.T) {
	s := openTemp(t)
	day := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	s.SaveState("t", reading(day, 12.5))
	ts, err := s.TrackBetween(day.Unix()-1, day.Unix()+1)
	if err != nil || len(ts) != 1 || ts[0].Lat == nil || *ts[0].Lat != 10 || *ts[0].Lon != 20 {
		t.Fatalf("%+v %v", ts, err)
	}
	if string(ts[0].Extra) != "{}" {
		t.Fatalf("the raw fields must not be read for a route: %s", ts[0].Extra)
	}
}
