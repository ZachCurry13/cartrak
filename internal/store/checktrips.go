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
