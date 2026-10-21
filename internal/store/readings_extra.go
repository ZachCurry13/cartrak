package store

// TirePoints returns the tire pressures (kilopascals) and outside temperature of every reading since `from` that had
// tire pressures, oldest first. Narrow columns only.
func (s *Store) TirePoints(from int64) ([]struct {
	TS       int64
	P        [4]*float64
	OutsideC *float64
}, error) {
	rows, err := s.DB.Query(`SELECT ts,tire_fl,tire_fr,tire_rl,tire_rr,outside_c FROM samples WHERE ts>=? AND tire_fl IS NOT NULL ORDER BY ts`, from)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []struct {
		TS       int64
		P        [4]*float64
		OutsideC *float64
	}
	for rows.Next() {
		var x struct {
			TS       int64
			P        [4]*float64
			OutsideC *float64
		}
		if err := rows.Scan(&x.TS, &x.P[0], &x.P[1], &x.P[2], &x.P[3], &x.OutsideC); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

// TripReadings returns every reading from `from` to `to` with only what explaining a trip needs.
func (s *Store) TripReadings(from, to int64) ([]struct {
	TS                              int64
	Odo, Kwh, Speed, Temp, Lat, Lon *float64
}, error) {
	rows, err := s.DB.Query(`SELECT ts,odometer_km,energy_kwh,speed_kph,outside_c,lat,lon FROM samples WHERE ts BETWEEN ? AND ? ORDER BY ts`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []struct {
		TS                              int64
		Odo, Kwh, Speed, Temp, Lat, Lon *float64
	}
	for rows.Next() {
		var x struct {
			TS                              int64
			Odo, Kwh, Speed, Temp, Lat, Lon *float64
		}
		if err := rows.Scan(&x.TS, &x.Odo, &x.Kwh, &x.Speed, &x.Temp, &x.Lat, &x.Lon); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

// EachExtra calls fn with the time and raw fields (JSON) of every reading from `from` to `to`, oldest first, one at a time: the raw
// fields are large (about 50 KB a reading), so a long stretch is never held in memory. Readings whose detail was removed to save
// space are skipped. fn stops the walk by returning an error.
func (s *Store) EachExtra(from, to int64, fn func(ts int64, extra []byte) error) error {
	rows, err := s.DB.Query(`SELECT ts,extra FROM samples WHERE ts BETWEEN ? AND ? AND extra IS NOT NULL AND length(extra)>2 ORDER BY ts`, from, to)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var ts int64
		var extra []byte
		if err := rows.Scan(&ts, &extra); err != nil {
			return err
		}
		if err := fn(ts, extra); err != nil {
			return err
		}
	}
	return rows.Err()
}
