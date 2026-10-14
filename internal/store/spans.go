package store

// SpanTrips returns the trips that ended after `from` (or are still going), oldest first, with only what is needed
// to say where and when the car moved. It has no row limit.
func (s *Store) SpanTrips(from int64) ([]*Trip, error) {
	rows, err := s.DB.Query(`SELECT id,start_ts,end_ts,end_lat,end_lon,open FROM trips WHERE end_ts>=? OR open=1 ORDER BY start_ts`, from)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Trip
	for rows.Next() {
		var t Trip
		if err := rows.Scan(&t.ID, &t.StartTS, &t.EndTS, &t.EndLat, &t.EndLon, &t.Open); err != nil {
			return nil, err
		}
		if t.Open {
			t.EndTS = nil
		}
		out = append(out, &t)
	}
	return out, rows.Err()
}

// SpanCharges returns the charging sessions that ended after `from` (or are still going), oldest first.
func (s *Store) SpanCharges(from int64) ([]*Charge, error) {
	rows, err := s.DB.Query(`SELECT id,start_ts,end_ts,open FROM charges WHERE end_ts>=? OR open=1 ORDER BY start_ts`, from)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Charge
	for rows.Next() {
		var c Charge
		if err := rows.Scan(&c.ID, &c.StartTS, &c.EndTS, &c.Open); err != nil {
			return nil, err
		}
		if c.Open {
			c.EndTS = nil
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}
