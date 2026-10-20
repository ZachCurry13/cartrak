// Package store persists everything cartrak learns, in SQLite.
package store

import (
	"database/sql"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type Store struct {
	DB   *sql.DB
	Path string // the database file
}

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
CREATE TABLE IF NOT EXISTS shares (
  id INTEGER PRIMARY KEY, kind TEXT NOT NULL, target_id INTEGER NOT NULL, title TEXT NOT NULL DEFAULT '',
  mode TEXT NOT NULL, show_time INTEGER NOT NULL DEFAULT 0, units TEXT NOT NULL DEFAULT 'mi',
  origin TEXT NOT NULL DEFAULT '', dest TEXT NOT NULL DEFAULT '', bbox TEXT NOT NULL DEFAULT '',
  created INTEGER NOT NULL, expires INTEGER NOT NULL DEFAULT 0, views INTEGER NOT NULL DEFAULT 0, last_view INTEGER NOT NULL DEFAULT 0,
  revoked INTEGER NOT NULL DEFAULT 0, token_hash TEXT UNIQUE NOT NULL, show_cities INTEGER NOT NULL DEFAULT 0);
CREATE TABLE IF NOT EXISTS maintenance (
  id INTEGER PRIMARY KEY, name TEXT NOT NULL, every_km REAL, every_days INTEGER, last_odo_km REAL,
  last_ts INTEGER NOT NULL DEFAULT 0, notified_ts INTEGER NOT NULL DEFAULT 0);
CREATE TABLE IF NOT EXISTS named_places (
  id INTEGER PRIMARY KEY, name TEXT NOT NULL, lat REAL NOT NULL, lon REAL NOT NULL, radius_m REAL NOT NULL DEFAULT 150);
CREATE TABLE IF NOT EXISTS parked_events (
  id INTEGER PRIMARY KEY, ts INTEGER NOT NULL, kind TEXT NOT NULL, value REAL NOT NULL DEFAULT 0, note TEXT NOT NULL DEFAULT '');
CREATE TABLE IF NOT EXISTS alert_log (
  id INTEGER PRIMARY KEY, ts INTEGER NOT NULL, kind TEXT NOT NULL, title TEXT NOT NULL DEFAULT '', body TEXT NOT NULL DEFAULT '',
  wanted INTEGER NOT NULL DEFAULT 0, sent INTEGER NOT NULL DEFAULT 0, quiet INTEGER NOT NULL DEFAULT 0, devices INTEGER NOT NULL DEFAULT 0);
CREATE TABLE IF NOT EXISTS journeys (
  id INTEGER PRIMARY KEY, name TEXT NOT NULL, created INTEGER NOT NULL);
CREATE TABLE IF NOT EXISTS signins (
  id INTEGER PRIMARY KEY, ts INTEGER NOT NULL, name TEXT NOT NULL, ok INTEGER NOT NULL,
  ip TEXT NOT NULL DEFAULT '', agent TEXT NOT NULL DEFAULT '');

CREATE TABLE IF NOT EXISTS devices (
  id INTEGER PRIMARY KEY, name TEXT NOT NULL, token_hash TEXT UNIQUE NOT NULL,
  created INTEGER NOT NULL, last_seen INTEGER NOT NULL DEFAULT 0, last_event TEXT NOT NULL DEFAULT '');
CREATE TABLE IF NOT EXISTS track_points (
  id INTEGER PRIMARY KEY, device_id INTEGER NOT NULL, ts INTEGER NOT NULL,
  lat REAL NOT NULL, lon REAL NOT NULL, speed_kph REAL, acc_m REAL);
CREATE INDEX IF NOT EXISTS track_points_ts ON track_points(ts);
CREATE TABLE IF NOT EXISTS places (k TEXT PRIMARY KEY, city TEXT NOT NULL, ts INTEGER NOT NULL);
CREATE TABLE IF NOT EXISTS trends (
  id INTEGER PRIMARY KEY, ts INTEGER NOT NULL, battery12v REAL, soc REAL, hv_volts REAL,
  capacity_kwh REAL, oil_pct REAL, fuel_pct REAL);
CREATE INDEX IF NOT EXISTS trends_ts ON trends(ts);
CREATE TABLE IF NOT EXISTS push_subs (
  id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL, endpoint TEXT UNIQUE NOT NULL,
  p256dh TEXT NOT NULL, auth TEXT NOT NULL, label TEXT NOT NULL DEFAULT '',
  events TEXT NOT NULL DEFAULT '', quiet TEXT NOT NULL DEFAULT '',
  created INTEGER NOT NULL, last_ok INTEGER NOT NULL DEFAULT 0, last_err TEXT NOT NULL DEFAULT '');
CREATE TABLE IF NOT EXISTS device_events (
  id INTEGER PRIMARY KEY, device_id INTEGER NOT NULL, ts INTEGER NOT NULL,
  type TEXT NOT NULL, lat REAL, lon REAL);
CREATE INDEX IF NOT EXISTS device_events_ts ON device_events(ts);
`

// Open opens (or creates) a car's database in dir.
func Open(dir string) (*Store, error) { return openFile(dir, "cartrak.db") }

func openFile(dir, name string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, name)
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path)+
		"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1) // SQLite: serialize writers, avoids SQLITE_BUSY
	if _, err := db.Exec(schema + schemaCars); err != nil {
		return nil, err
	}
	if err := migrate(db); err != nil {
		return nil, err
	}
	return &Store{DB: db, Path: path}, nil
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
