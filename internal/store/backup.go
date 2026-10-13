package store

import (
	"database/sql"
	"path/filepath"
)

// DBSize is the database size in bytes, from SQLite's own page count.
func (s *Store) DBSize() int64 {
	var n int64
	_ = s.DB.QueryRow(`SELECT page_count*page_size FROM pragma_page_count(), pragma_page_size()`).Scan(&n)
	return n
}

// Backup writes a consistent copy of the database to dest (which must not exist). Sign-in sessions are
// always left out, so a backup cannot be used to get into a running CarTrak. A small backup also leaves out
// the raw copies of every Ford reading (the biggest part of the database); trips, charging, places, trends,
// positions, users and settings are all kept.
func (s *Store) Backup(dest string, small bool) error {
	if _, err := s.DB.Exec(`VACUUM INTO ?`, filepath.ToSlash(dest)); err != nil {
		return err
	}
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(dest))
	if err != nil {
		return err
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`DELETE FROM sessions`); err != nil {
		return err
	}
	if !small {
		return nil
	}
	if _, err := db.Exec(`DELETE FROM snapshots; UPDATE samples SET extra='{}'`); err != nil {
		return err
	}
	_, err = db.Exec(`VACUUM`)
	return err
}
