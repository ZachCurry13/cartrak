package store

import (
	"database/sql"
	"path/filepath"
	"testing"
)

// An install created before trips had a time zone must upgrade cleanly and keep its data.
func TestMigrationAddsTripTimeZone(t *testing.T) {
	dir := t.TempDir()
	old, err := sql.Open("sqlite", "file:"+filepath.ToSlash(filepath.Join(dir, "cartrak.db")))
	if err != nil {
		t.Fatal(err)
	}
	_, err = old.Exec(`CREATE TABLE trips (id INTEGER PRIMARY KEY, start_ts INTEGER NOT NULL, end_ts INTEGER,
		start_lat REAL, start_lon REAL, end_lat REAL, end_lon REAL, start_odo REAL, end_odo REAL, distance_km REAL DEFAULT 0,
		start_soc REAL, end_soc REAL, max_speed_kph REAL DEFAULT 0, category TEXT NOT NULL DEFAULT 'personal',
		category_source TEXT NOT NULL DEFAULT 'default', rule_id INTEGER, notes TEXT DEFAULT '', open INTEGER NOT NULL DEFAULT 1);
		INSERT INTO trips(start_ts, category) VALUES (1700000000, 'work');`)
	if err != nil {
		t.Fatal(err)
	}
	old.Close()

	s, err := Open(dir)
	if err != nil {
		t.Fatalf("upgrade failed: %v", err)
	}
	ts, err := s.ListTrips(TripFilter{})
	if err != nil || len(ts) != 1 || ts[0].Category != "work" || ts[0].TZ != "" {
		t.Fatalf("old trip lost or changed: %v %+v", err, ts)
	}
	// Opening again does nothing (already migrated), and new trips can store a zone.
	s.Close()
	s, err = Open(dir)
	if err != nil {
		t.Fatalf("second open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	tr := &Trip{StartTS: 1700001000, Category: "personal", CategorySource: "default", TZ: "America/New_York"}
	if err := s.CreateTrip(tr); err != nil {
		t.Fatal(err)
	}
	got, _ := s.GetTrip(tr.ID)
	if got.TZ != "America/New_York" {
		t.Fatalf("zone not stored: %q", got.TZ)
	}
}

// Every column a later version added must be added again to a database made before it, with the data kept. The test makes that
// database by dropping each migrated column from a current one, then opening it the way an upgrade would.
func TestEveryMigrationUpgradesADatabaseThatLacksItsColumn(t *testing.T) {
	skipped := 0
	for _, m := range migrations {
		dir := t.TempDir()
		s, err := Open(dir)
		if err != nil {
			t.Fatal(err)
		}
		if err := s.CreateTrip(&Trip{StartTS: 1700000000, Category: "work", CategorySource: "default"}); err != nil {
			t.Fatal(err)
		}
		if _, err := s.DB.Exec(`ALTER TABLE ` + m.table + ` DROP COLUMN ` + m.column); err != nil {
			skipped++ // SQLite cannot drop an indexed column; that one is covered by the older-schema test above
			s.Close()
			continue
		}
		s.Close()
		up, err := Open(dir)
		if err != nil {
			t.Fatalf("upgrading a database without %s.%s: %v", m.table, m.column, err)
		}
		if has, err := hasColumn(up.DB, m.table, m.column); err != nil || !has {
			t.Fatalf("%s.%s was not added back: %v", m.table, m.column, err)
		}
		if ts, err := up.ListTrips(TripFilter{}); err != nil || len(ts) != 1 || ts[0].Category != "work" {
			t.Fatalf("the data survived the upgrade of %s.%s: %v %+v", m.table, m.column, err, ts)
		}
		up.Close()
	}
	if skipped == len(migrations) {
		t.Fatal("no column could be dropped, so nothing was tested")
	}
}
