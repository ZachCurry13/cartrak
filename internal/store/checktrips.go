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
