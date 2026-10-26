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

// fillTripStarts gives the trips that have no start position the place the car was last seen before they began (where it was last
// parked), the same rule new trips follow. It runs once for each database.
func (s *Store) fillTripStarts() {
	const flag = "trip_start_filled_v1"
	if s.GetKV(flag) != "" {
		return
	}
	_, err := s.DB.Exec(`UPDATE trips SET
		start_lat = (SELECT lat FROM samples WHERE ts < trips.start_ts AND lat IS NOT NULL AND lon IS NOT NULL ORDER BY ts DESC LIMIT 1),
		start_lon = (SELECT lon FROM samples WHERE ts < trips.start_ts AND lat IS NOT NULL AND lon IS NOT NULL ORDER BY ts DESC LIMIT 1)
		WHERE (start_lat IS NULL OR start_lon IS NULL)`)
	if err == nil {
		_ = s.SetKV(flag, "1")
	}
}
