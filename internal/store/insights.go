package store

// EnergyTrips returns the finished trips since `from` with just the fields the efficiency insights need, newest
// first, without the usual row limit.
func (s *Store) EnergyTrips(from int64) ([]*Trip, error) {
	rows, err := s.DB.Query(`SELECT id,start_ts,end_ts,distance_km,start_soc,end_soc,start_kwh,end_kwh FROM trips
		WHERE open=0 AND end_ts IS NOT NULL AND start_ts>=? ORDER BY start_ts DESC`, from)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Trip
	for rows.Next() {
		var t Trip
		if err := rows.Scan(&t.ID, &t.StartTS, &t.EndTS, &t.DistanceKm, &t.StartSoC, &t.EndSoC, &t.StartKwh, &t.EndKwh); err != nil {
			return nil, err
		}
		out = append(out, &t)
	}
	return out, rows.Err()
}

// TripTemps is the average outside temperature (Celsius) during each trip since `from`, from the small trends
// table. Trips with no temperature readings are left out.
func (s *Store) TripTemps(from int64) (map[int64]float64, error) {
	rows, err := s.DB.Query(`SELECT t.id,AVG(r.outside_c) FROM trips t JOIN trends r ON r.ts BETWEEN t.start_ts AND t.end_ts
		WHERE t.open=0 AND t.start_ts>=? AND r.outside_c IS NOT NULL GROUP BY t.id`, from)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]float64{}
	for rows.Next() {
		var id int64
		var avg float64
		if err := rows.Scan(&id, &avg); err != nil {
			return nil, err
		}
		out[id] = avg
	}
	return out, rows.Err()
}
