package store

// Via is a place the owner says a trip went through, so its estimated road can be drawn through it (the car recorded too few
// positions to show a detour).
type Via struct {
	Name string  `json:"name"`
	Lat  float64 `json:"lat"`
	Lon  float64 `json:"lon"`
}

const tripViaDDL = `CREATE TABLE IF NOT EXISTS trip_via (
 trip_id INTEGER NOT NULL, seq INTEGER NOT NULL, name TEXT NOT NULL DEFAULT '', lat REAL NOT NULL, lon REAL NOT NULL,
 PRIMARY KEY (trip_id, seq))`

// TripVia lists the stops added to a trip, in the order they were added.
func (s *Store) TripVia(tripID int64) ([]Via, error) {
	if _, err := s.DB.Exec(tripViaDDL); err != nil {
		return nil, err
	}
	rows, err := s.DB.Query(`SELECT name,lat,lon FROM trip_via WHERE trip_id=? ORDER BY seq`, tripID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Via{}
	for rows.Next() {
		var v Via
		if err := rows.Scan(&v.Name, &v.Lat, &v.Lon); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// SetTripVia replaces a trip's stops with the given ones (none removes them all).
func (s *Store) SetTripVia(tripID int64, via []Via) error {
	if _, err := s.DB.Exec(tripViaDDL); err != nil {
		return err
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM trip_via WHERE trip_id=?`, tripID); err != nil {
		return err
	}
	for i, v := range via {
		if _, err := tx.Exec(`INSERT INTO trip_via(trip_id,seq,name,lat,lon) VALUES(?,?,?,?,?)`, tripID, i, v.Name, v.Lat, v.Lon); err != nil {
			return err
		}
	}
	return tx.Commit()
}
