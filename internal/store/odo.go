package store

// OdoPoint is one odometer reading (kilometers) and when the car reported it.
type OdoPoint struct {
	TS int64
	Km float64
}

// OdoBetween returns the odometer readings from `from` up to (not including) `to`, preceded by the last
// reading before `from`, so the first point is where the car stood when the period began. It reads only
// the small trends table.
func (s *Store) OdoBetween(from, to int64) ([]OdoPoint, error) {
	var out []OdoPoint
	var base OdoPoint
	err := s.DB.QueryRow(`SELECT ts,odometer_km FROM trends WHERE odometer_km IS NOT NULL AND ts<? ORDER BY ts DESC LIMIT 1`, from).Scan(&base.TS, &base.Km)
	if err == nil {
		out = append(out, base)
	}
	rows, err := s.DB.Query(`SELECT ts,odometer_km FROM trends WHERE odometer_km IS NOT NULL AND ts>=? AND ts<? ORDER BY ts`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var p OdoPoint
		if err := rows.Scan(&p.TS, &p.Km); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// TripsOverlappingOdo returns the trips whose odometer range touches [lo, hi]. Only the fields the
// mileage check needs are filled in.
func (s *Store) TripsOverlappingOdo(lo, hi float64) ([]*Trip, error) {
	rows, err := s.DB.Query(`SELECT id,start_ts,end_ts,start_odo,end_odo,distance_km FROM trips
		WHERE start_odo IS NOT NULL AND end_odo IS NOT NULL AND end_odo>=? AND start_odo<=? ORDER BY start_odo`, lo, hi)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Trip
	for rows.Next() {
		var t Trip
		if err := rows.Scan(&t.ID, &t.StartTS, &t.EndTS, &t.StartOdo, &t.EndOdo, &t.DistanceKm); err != nil {
			return nil, err
		}
		out = append(out, &t)
	}
	return out, rows.Err()
}

// AddManualTrip stores a finished trip the owner added from the mileage check. The category is the
// owner's own choice, so it is marked manual and rules never change it.
func (s *Store) AddManualTrip(t *Trip) error {
	r, err := s.DB.Exec(`INSERT INTO trips(start_ts,end_ts,start_lat,start_lon,end_lat,end_lon,start_odo,end_odo,distance_km,
		start_soc,end_soc,category,category_source,notes,open,tz) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,'manual',?,0,?)`,
		t.StartTS, t.EndTS, t.StartLat, t.StartLon, t.EndLat, t.EndLon, t.StartOdo, t.EndOdo, t.DistanceKm,
		t.StartSoC, t.EndSoC, t.Category, t.Notes, t.TZ)
	if err != nil {
		return err
	}
	t.ID, _ = r.LastInsertId()
	return nil
}

// SetTripOdometer widens a trip's odometer range and distance, for miles the car drove just before or
// after the part CarTrak recorded.
// SetTripDistance sets a finished trip's distance and says where the figure came from ("road", "manual", or "" to go back to what the
// odometer said).
func (s *Store) SetTripDistance(id int64, km float64, src string) error {
	_, err := s.DB.Exec(`UPDATE trips SET distance_km=?,distance_src=? WHERE id=?`, km, src, id)
	return err
}

func (s *Store) SetTripOdometer(id int64, startOdo, endOdo, distanceKm float64) error {
	_, err := s.DB.Exec(`UPDATE trips SET start_odo=?,end_odo=?,distance_km=? WHERE id=?`, startOdo, endOdo, distanceKm, id)
	return err
}
