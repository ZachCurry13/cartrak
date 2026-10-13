// Package store persists everything cartrak learns, in SQLite.
package store

import (
	"database/sql"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type Store struct{ DB *sql.DB }

const schema = `
CREATE TABLE IF NOT EXISTS snapshots (
  id INTEGER PRIMARY KEY, ts INTEGER NOT NULL, provider TEXT, raw TEXT NOT NULL);
CREATE INDEX IF NOT EXISTS snapshots_ts ON snapshots(ts);

CREATE TABLE IF NOT EXISTS samples (
  id INTEGER PRIMARY KEY, ts INTEGER NOT NULL, snapshot_id INTEGER,
  odometer_km REAL, soc_pct REAL, range_km REAL, lat REAL, lon REAL,
  speed_kph REAL, ignition TEXT, charge_state TEXT, plug TEXT,
  charge_kw REAL, outside_c REAL, battery12v_v REAL,
  tire_fl REAL, tire_fr REAL, tire_rl REAL, tire_rr REAL,
  locked INTEGER, extra TEXT);
CREATE INDEX IF NOT EXISTS samples_ts ON samples(ts);

CREATE TABLE IF NOT EXISTS trips (
  id INTEGER PRIMARY KEY,
  start_ts INTEGER NOT NULL, end_ts INTEGER,
  start_lat REAL, start_lon REAL, end_lat REAL, end_lon REAL,
  start_odo REAL, end_odo REAL, distance_km REAL DEFAULT 0,
  start_soc REAL, end_soc REAL, max_speed_kph REAL DEFAULT 0,
  category TEXT NOT NULL DEFAULT 'personal',
  category_source TEXT NOT NULL DEFAULT 'default',
  rule_id INTEGER, notes TEXT DEFAULT '', open INTEGER NOT NULL DEFAULT 1,
  tz TEXT NOT NULL DEFAULT '');
CREATE INDEX IF NOT EXISTS trips_start ON trips(start_ts);

CREATE TABLE IF NOT EXISTS charges (
  id INTEGER PRIMARY KEY, start_ts INTEGER NOT NULL, end_ts INTEGER,
  lat REAL, lon REAL, start_soc REAL, end_soc REAL,
  peak_kw REAL DEFAULT 0, open INTEGER NOT NULL DEFAULT 1);

CREATE TABLE IF NOT EXISTS rules (
  id INTEGER PRIMARY KEY, name TEXT NOT NULL, category TEXT NOT NULL,
  priority INTEGER NOT NULL DEFAULT 0, enabled INTEGER NOT NULL DEFAULT 1,
  days INTEGER NOT NULL DEFAULT 62,      -- bitmask Sun=1 Mon=2 ... Sat=64
  start_min INTEGER NOT NULL DEFAULT 480, end_min INTEGER NOT NULL DEFAULT 1020,
  geo_lat REAL, geo_lon REAL, geo_radius_m REAL);

CREATE TABLE IF NOT EXISTS users (
  id INTEGER PRIMARY KEY, name TEXT UNIQUE NOT NULL, hash TEXT NOT NULL,
  role TEXT NOT NULL DEFAULT 'viewer');
CREATE TABLE IF NOT EXISTS sessions (
  token TEXT PRIMARY KEY, user_id INTEGER NOT NULL, expires INTEGER NOT NULL);
CREATE TABLE IF NOT EXISTS kv (k TEXT PRIMARY KEY, v TEXT NOT NULL);
`

func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(filepath.Join(dir, "cartrak.db"))+
		"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1) // SQLite: serialize writers, avoids SQLITE_BUSY
	if _, err := db.Exec(schema); err != nil {
		return nil, err
	}
	if err := migrate(db); err != nil {
		return nil, err
	}
	return &Store{DB: db}, nil
}

func (s *Store) GetKV(k string) string {
	var v string
	_ = s.DB.QueryRow(`SELECT v FROM kv WHERE k=?`, k).Scan(&v)
	return v
}

func (s *Store) SetKV(k, v string) error {
	_, err := s.DB.Exec(`INSERT INTO kv(k,v) VALUES(?,?) ON CONFLICT(k) DO UPDATE SET v=excluded.v`, k, v)
	return err
}

func (s *Store) Close() error { return s.DB.Close() }
