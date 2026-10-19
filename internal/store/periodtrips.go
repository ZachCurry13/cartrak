package store

// PeriodTrip is a trip reduced to what the statistics need.
type PeriodTrip struct {
	Start      int64
	End        int64
	DistanceKm float64
}

// PeriodTrips returns the finished trips that started in from..to, with no row limit.
func (s *Store) PeriodTrips(from, to int64) ([]PeriodTrip, error) {
	rows, err := s.DB.Query(`SELECT start_ts,COALESCE(end_ts,start_ts),distance_km FROM trips WHERE open=0 AND start_ts>=? AND start_ts<? ORDER BY start_ts`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PeriodTrip
	for rows.Next() {
		var t PeriodTrip
		if err := rows.Scan(&t.Start, &t.End, &t.DistanceKm); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
