package store

// A person's label and note for a stop: the parked time after a trip. The stop itself is never stored, only what was said about it,
// against the trip that arrived there.

// StopLabel is a category chosen by hand ("" when only a note was written) and a note such as a customer's name.
type StopLabel struct {
	Category string
	Note     string
}

// StopLabels is every label, by the trip that arrived at the stop.
func (s *Store) StopLabels() (map[int64]StopLabel, error) {
	rows, err := s.DB.Query(`SELECT trip_id, category, note FROM stop_labels`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]StopLabel{}
	for rows.Next() {
		var id int64
		var l StopLabel
		if err := rows.Scan(&id, &l.Category, &l.Note); err != nil {
			return nil, err
		}
		out[id] = l
	}
	return out, rows.Err()
}

// SetStopLabel saves what was said about a stop. A label with neither a category nor a note is removed.
func (s *Store) SetStopLabel(tripID int64, l StopLabel) error {
	if l.Category == "" && l.Note == "" {
		_, err := s.DB.Exec(`DELETE FROM stop_labels WHERE trip_id=?`, tripID)
		return err
	}
	_, err := s.DB.Exec(`INSERT INTO stop_labels(trip_id,category,note) VALUES(?,?,?)
		ON CONFLICT(trip_id) DO UPDATE SET category=excluded.category, note=excluded.note`, tripID, l.Category, l.Note)
	return err
}
