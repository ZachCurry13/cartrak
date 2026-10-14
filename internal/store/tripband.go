package store

// BandTrips returns the finished trips since `from` that have battery readings at both ends, newest first.
// It sets only what the range estimate needs (start, time zone, distance and the two percentages) and,
// unlike ListTrips, has no row limit, so a heavy driver's whole year is counted.
func (s *Store) BandTrips(from int64) ([]*Trip, error) {
	rows, err := s.DB.Query(`SELECT id,start_ts,tz,distance_km,start_soc,end_soc FROM trips
		WHERE open=0 AND start_ts>=? AND start_soc IS NOT NULL AND end_soc IS NOT NULL ORDER BY start_ts DESC`, from)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Trip
	for rows.Next() {
		var t Trip
		if err := rows.Scan(&t.ID, &t.StartTS, &t.TZ, &t.DistanceKm, &t.StartSoC, &t.EndSoC); err != nil {
			return nil, err
		}
		out = append(out, &t)
	}
	return out, rows.Err()
}
