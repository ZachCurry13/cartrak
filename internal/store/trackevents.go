package store

import "time"

const trackDDL = `CREATE TABLE IF NOT EXISTS track_events (id INTEGER PRIMARY KEY, ts INTEGER NOT NULL, kind TEXT NOT NULL,
	a TEXT NOT NULL DEFAULT '', b TEXT NOT NULL DEFAULT '', n INTEGER NOT NULL DEFAULT 0);
	CREATE INDEX IF NOT EXISTS track_events_ts ON track_events(ts)`

// TrackEvent is one thing that happened while tracking was on: a request the app made (kind "request": a is the method and path,
// b the status, n the milliseconds), a page opened, a button tapped or an error on the phone (kinds "page", "tap", "error": a is
// what, b is the page it was on).
type TrackEvent struct {
	TS   int64  `json:"ts"`
	Kind string `json:"kind"`
	A    string `json:"a"`
	B    string `json:"b"`
	N    int64  `json:"n"`
}

const trackKeepDays = 14

// AddTrackEvents keeps events, and forgets the ones older than two weeks.
func (s *Store) AddTrackEvents(es []TrackEvent) error {
	if _, err := s.DB.Exec(trackDDL); err != nil {
		return err
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	for _, e := range es {
		if _, err := tx.Exec(`INSERT INTO track_events(ts,kind,a,b,n) VALUES(?,?,?,?,?)`, e.TS, e.Kind, e.A, e.B, e.N); err != nil {
			tx.Rollback()
			return err
		}
	}
	_, _ = tx.Exec(`DELETE FROM track_events WHERE ts < ?`, time.Now().AddDate(0, 0, -trackKeepDays).Unix())
	return tx.Commit()
}

// TrackEventsBetween are the events in [from, to], oldest first.
func (s *Store) TrackEventsBetween(from, to int64) ([]TrackEvent, error) {
	if _, err := s.DB.Exec(trackDDL); err != nil {
		return nil, err
	}
	rows, err := s.DB.Query(`SELECT ts,kind,a,b,n FROM track_events WHERE ts BETWEEN ? AND ? ORDER BY ts,id`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TrackEvent
	for rows.Next() {
		var e TrackEvent
		if err := rows.Scan(&e.TS, &e.Kind, &e.A, &e.B, &e.N); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
