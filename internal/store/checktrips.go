package store

// CheckTrips returns the trips that started after `from`, with only what the data check needs. It has no row limit.
func (s *Store) CheckTrips(from int64) ([]*Trip, error) {
	rows, err := s.DB.Query(`SELECT id,start_ts,end_ts,distance_km,open FROM trips WHERE start_ts>=? ORDER BY start_ts`, from)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Trip
	for rows.Next() {
		var t Trip
		if err := rows.Scan(&t.ID, &t.StartTS, &t.EndTS, &t.DistanceKm, &t.Open); err != nil {
			return nil, err
		}
		out = append(out, &t)
	}
	return out, rows.Err()
}

// CloseTrip marks a trip that never ended as finished, keeping the last position and time it recorded.
// It reports false when the trip is not open.
func (s *Store) CloseTrip(id int64) (bool, error) {
	r, err := s.DB.Exec(`UPDATE trips SET open=0 WHERE id=? AND open=1`, id)
	if err != nil {
		return false, err
	}
	n, _ := r.RowsAffected()
	return n > 0, nil
}

// DeleteTripAndShares removes a trip and any public link made from it, so no link points at nothing.
func (s *Store) DeleteTripAndShares(id int64) error {
	if _, err := s.DB.Exec(`DELETE FROM shares WHERE kind='trip' AND target_id=?`, id); err != nil {
		return err
	}
	return s.DeleteTrip(id)
}

// EmptyTrips returns the ids of finished trips that went nowhere: under 0.2 km, never faster than 8 km/h, not part of
// a journey and with no note. These come from sitting in the car with the ignition on.
func (s *Store) EmptyTrips() ([]int64, error) {
	rows, err := s.DB.Query(`SELECT id FROM trips WHERE open=0 AND distance_km<0.2 AND max_speed_kph<8 AND journey_id IS NULL AND COALESCE(notes,'')=''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
