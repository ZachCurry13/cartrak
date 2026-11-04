package store

// DailyHighs is the highest outside temperature (Celsius) the car reported on each UTC day since from, oldest first, as day number
// (days since 1970) and degrees. Days with no temperature are left out.
func (s *Store) DailyHighs(from int64) (days []int64, highs []float64, err error) {
	rows, err := s.DB.Query(`SELECT ts/86400, MAX(outside_c) FROM samples WHERE ts>=? AND outside_c IS NOT NULL GROUP BY ts/86400 ORDER BY ts/86400`, from)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var d int64
		var h float64
		if err := rows.Scan(&d, &h); err != nil {
			return nil, nil, err
		}
		days, highs = append(days, d), append(highs, h)
	}
	return days, highs, rows.Err()
}
