package store

import (
	"database/sql"
	"time"
)

const tripShareDDL = `CREATE TABLE IF NOT EXISTS trip_shares (trip_id INTEGER PRIMARY KEY, token TEXT NOT NULL, token_hash TEXT NOT NULL, expires INTEGER NOT NULL,
 hide_start INTEGER NOT NULL DEFAULT 1, created INTEGER NOT NULL, opens INTEGER NOT NULL DEFAULT 0)`

// TripShare is the link that lets someone read a saved trip without signing in.
type TripShare struct {
	TripID    int64  `json:"tripId"`
	Token     string `json:"-"`
	Expires   int64  `json:"expires"`
	HideStart bool   `json:"hideStart"`
	Created   int64  `json:"created"`
	Opens     int    `json:"opens"`
}

func (s *Store) tripShareInit() error { _, err := s.DB.Exec(tripShareDDL); return err }

// SetTripShare makes (or replaces) the share link of a saved trip.
func (s *Store) SetTripShare(tripID int64, token, hash string, expires int64, hideStart bool) error {
	if err := s.tripShareInit(); err != nil {
		return err
	}
	hide := 0
	if hideStart {
		hide = 1
	}
	_, err := s.DB.Exec(`INSERT INTO trip_shares(trip_id,token,token_hash,expires,hide_start,created) VALUES(?,?,?,?,?,?)
		ON CONFLICT(trip_id) DO UPDATE SET token=excluded.token, token_hash=excluded.token_hash, expires=excluded.expires, hide_start=excluded.hide_start, created=excluded.created, opens=0`,
		tripID, token, hash, expires, hide, time.Now().Unix())
	return err
}

// ClearTripShare stops sharing a saved trip.
func (s *Store) ClearTripShare(tripID int64) error {
	if err := s.tripShareInit(); err != nil {
		return err
	}
	_, err := s.DB.Exec(`DELETE FROM trip_shares WHERE trip_id=?`, tripID)
	return err
}

// TripShares lists the links in use, by trip.
func (s *Store) TripShares() (map[int64]TripShare, error) {
	if err := s.tripShareInit(); err != nil {
		return nil, err
	}
	rows, err := s.DB.Query(`SELECT trip_id,token,expires,hide_start,created,opens FROM trip_shares`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]TripShare{}
	for rows.Next() {
		var t TripShare
		var hide int
		if err := rows.Scan(&t.TripID, &t.Token, &t.Expires, &hide, &t.Created, &t.Opens); err != nil {
			return nil, err
		}
		t.HideStart = hide == 1
		out[t.TripID] = t
	}
	return out, rows.Err()
}

// TripByShare is the saved trip a link opens, while the link has not expired; it counts the opening.
func (s *Store) TripByShare(hash string, now int64) (*SavedTrip, bool, error) {
	if err := s.tripShareInit(); err != nil {
		return nil, false, err
	}
	if _, err := s.DB.Exec(savedTripDDL); err != nil {
		return nil, false, err
	}
	var t SavedTrip
	var hide int
	err := s.DB.QueryRow(`SELECT t.id,t.name,t.body,t.created,h.hide_start FROM trip_shares h JOIN saved_trips t ON t.id=h.trip_id WHERE h.token_hash=? AND h.expires>?`, hash, now).
		Scan(&t.ID, &t.Name, &t.Body, &t.Created, &hide)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	_, _ = s.DB.Exec(`UPDATE trip_shares SET opens=opens+1 WHERE trip_id=?`, t.ID)
	return &t, hide == 1, nil
}
