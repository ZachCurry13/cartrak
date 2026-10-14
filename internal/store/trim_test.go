package store

import (
	"context"
	"testing"
	"time"
)

func TestTrimRemovesRawCopiesAndDetailOfOldReadingsOnlyAndKeepsTheNumbers(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for i := 0; i < 5; i++ {
		ts := int64(1000 + i*100) // old readings
		if i >= 3 {
			ts = 900000 + int64(i) // recent
		}
		r, err := s.DB.Exec(`INSERT INTO snapshots(ts,provider,raw) VALUES(?,?,?)`, ts, "t", `{"big":"payload"}`)
		if err != nil {
			t.Fatal(err)
		}
		id, _ := r.LastInsertId()
		if _, err := s.DB.Exec(`INSERT INTO samples(ts,snapshot_id,soc_pct,odometer_km,extra) VALUES(?,?,?,?,?)`, ts, id, 50.0, 1234.0, `{"detail":1}`); err != nil {
			t.Fatal(err)
		}
	}
	cutoff := int64(500000)
	if sn, sa, err := s.StorageCounts(cutoff); err != nil || sn != 3 || sa != 3 {
		t.Fatalf("three old of each: %d %d %v", sn, sa, err)
	}
	var last int
	if err := s.TrimRaw(context.Background(), cutoff, 2, time.Millisecond, func(n int) { last = n }); err != nil {
		t.Fatal(err)
	}
	if last != 6 {
		t.Fatalf("progress reaches every row touched: %d", last)
	}
	var snaps, oldDetail, newDetail int
	s.DB.QueryRow(`SELECT COUNT(*) FROM snapshots`).Scan(&snaps)
	s.DB.QueryRow(`SELECT COUNT(*) FROM samples WHERE extra='{}'`).Scan(&oldDetail)
	s.DB.QueryRow(`SELECT COUNT(*) FROM samples WHERE extra='{"detail":1}'`).Scan(&newDetail)
	if snaps != 2 || oldDetail != 3 || newDetail != 2 {
		t.Fatalf("snapshots left %d, emptied %d, untouched %d", snaps, oldDetail, newDetail)
	}
	var soc float64
	s.DB.QueryRow(`SELECT soc_pct FROM samples WHERE ts=1000`).Scan(&soc)
	if soc != 50 {
		t.Fatal("the reading's own numbers are kept")
	}
}
