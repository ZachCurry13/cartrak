package store

import "time"

// Point is one GPS fix sent by a phone. Speed is km/h, accuracy is meters.
type Point struct {
	TS       int64    `json:"ts"`
	Lat      float64  `json:"lat"`
	Lon      float64  `json:"lon"`
	SpeedKph *float64 `json:"speedKph,omitempty"`
	AccM     *float64 `json:"accM,omitempty"`
}

// SavePoints stores a batch of fixes in one transaction.
func (s *Store) SavePoints(device int64, pts []Point) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	st, err := tx.Prepare(`INSERT INTO track_points(device_id,ts,lat,lon,speed_kph,acc_m) VALUES(?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer st.Close()
	for _, p := range pts {
		if _, err := st.Exec(device, p.TS, p.Lat, p.Lon, p.SpeedKph, p.AccM); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// PointsBetween returns every phone fix from from to to (unix seconds), oldest first.
func (s *Store) PointsBetween(from, to int64) ([]Point, error) {
	rows, err := s.DB.Query(`SELECT ts,lat,lon,speed_kph,acc_m FROM track_points WHERE ts BETWEEN ? AND ? ORDER BY ts, id`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Point{}
	for rows.Next() {
		var p Point
		if err := rows.Scan(&p.TS, &p.Lat, &p.Lon, &p.SpeedKph, &p.AccM); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// DeletePointsBefore removes fixes older than the cutoff (the optional cleanup setting).
func (s *Store) DeletePointsBefore(cutoff time.Time) (int64, error) {
	r, err := s.DB.Exec(`DELETE FROM track_points WHERE ts < ?`, cutoff.Unix())
	if err != nil {
		return 0, err
	}
	return r.RowsAffected()
}

// Event is something a phone reported: drive_start, drive_end or ping.
type Event struct {
	ID     int64    `json:"id"`
	Device int64    `json:"device"`
	TS     int64    `json:"ts"`
	Type   string   `json:"type"`
	Lat    *float64 `json:"lat,omitempty"`
	Lon    *float64 `json:"lon,omitempty"`
}

func (s *Store) AddEvent(e Event) error {
	_, err := s.DB.Exec(`INSERT INTO device_events(device_id,ts,type,lat,lon) VALUES(?,?,?,?,?)`, e.Device, e.TS, e.Type, e.Lat, e.Lon)
	return err
}

// RecentEvents lists the newest phone events first (for "did my automation fire?").
func (s *Store) RecentEvents(limit int) ([]Event, error) {
	rows, err := s.DB.Query(`SELECT id,device_id,ts,type,lat,lon FROM device_events ORDER BY ts DESC, id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Event{}
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.ID, &e.Device, &e.TS, &e.Type, &e.Lat, &e.Lon); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// SampleCounts is how many readings the car sent between two times and how many different positions they carried. The
// car repeats its last known position in every reading, so the number that matters is how many distinct ones there are.
func (s *Store) SampleCounts(from, to int64) (total, distinctPositions int, err error) {
	err = s.DB.QueryRow(`SELECT COUNT(*), COUNT(DISTINCT CASE WHEN lat IS NOT NULL AND lon IS NOT NULL THEN printf('%.4f,%.4f', lat, lon) END) FROM samples WHERE ts BETWEEN ? AND ?`, from, to).Scan(&total, &distinctPositions)
	return
}
