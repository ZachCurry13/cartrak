package store

// SavedTrip is a planned long trip kept to open again: what was asked and what was planned, as the page wrote it.
type SavedTrip struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Body    string `json:"body"` // JSON from the page
	Created int64  `json:"created"`
}

const savedTripDDL = `CREATE TABLE IF NOT EXISTS saved_trips (id INTEGER PRIMARY KEY, name TEXT NOT NULL, body TEXT NOT NULL, created INTEGER NOT NULL)`

// SavedTrips lists the saved trips, newest first.
func (s *Store) SavedTrips() ([]SavedTrip, error) {
	if _, err := s.DB.Exec(savedTripDDL); err != nil {
		return nil, err
	}
	rows, err := s.DB.Query(`SELECT id,name,body,created FROM saved_trips ORDER BY created DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SavedTrip{}
	for rows.Next() {
		var t SavedTrip
		if err := rows.Scan(&t.ID, &t.Name, &t.Body, &t.Created); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// AddSavedTrip keeps a trip and returns it with its number.
func (s *Store) AddSavedTrip(name, body string, now int64) (SavedTrip, error) {
	if _, err := s.DB.Exec(savedTripDDL); err != nil {
		return SavedTrip{}, err
	}
	r, err := s.DB.Exec(`INSERT INTO saved_trips(name,body,created) VALUES(?,?,?)`, name, body, now)
	if err != nil {
		return SavedTrip{}, err
	}
	id, _ := r.LastInsertId()
	return SavedTrip{ID: id, Name: name, Body: body, Created: now}, nil
}

// DeleteSavedTrip forgets a saved trip.
func (s *Store) DeleteSavedTrip(id int64) error {
	if _, err := s.DB.Exec(savedTripDDL); err != nil {
		return err
	}
	_, err := s.DB.Exec(`DELETE FROM saved_trips WHERE id=?`, id)
	return err
}
