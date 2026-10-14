package store

// YearTrips returns the finished trips that started in [from, to), oldest first, with what a yearly summary needs:
// distance, category and the readings used to work out energy.
func (s *Store) YearTrips(from, to int64) ([]*Trip, error) {
	rows, err := s.DB.Query(`SELECT id,start_ts,end_ts,distance_km,category,start_soc,end_soc,start_kwh,end_kwh FROM trips
		WHERE open=0 AND end_ts IS NOT NULL AND start_ts>=? AND start_ts<? ORDER BY start_ts`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Trip
	for rows.Next() {
		var t Trip
		if err := rows.Scan(&t.ID, &t.StartTS, &t.EndTS, &t.DistanceKm, &t.Category, &t.StartSoC, &t.EndSoC, &t.StartKwh, &t.EndKwh); err != nil {
			return nil, err
		}
		out = append(out, &t)
	}
	return out, rows.Err()
}

// FirstTripTS is when the earliest recorded trip started, or 0 when there are none.
func (s *Store) FirstTripTS() int64 {
	var ts int64
	_ = s.DB.QueryRow(`SELECT COALESCE(MIN(start_ts),0) FROM trips`).Scan(&ts)
	return ts
}
