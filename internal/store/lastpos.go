package store

import "database/sql"

// LastPositionBefore is the newest position the car reported before ts: where it was last known to be, which for a trip that has no
// start position of its own is where it was last parked. It returns nils when there is none.
func (s *Store) LastPositionBefore(ts int64) (lat, lon *float64) {
	var la, lo float64
	err := s.DB.QueryRow(`SELECT lat, lon FROM samples WHERE ts < ? AND lat IS NOT NULL AND lon IS NOT NULL ORDER BY ts DESC LIMIT 1`, ts).Scan(&la, &lo)
	if err == sql.ErrNoRows || err != nil {
		return nil, nil
	}
	return &la, &lo
}
