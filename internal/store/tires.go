package store

import "strings"

// TireSet is a set of tires the car wore from a date until the next set went on.
type TireSet struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Kind   string `json:"kind"` // "winter", "summer", "allseason" or "other"
	FromTS int64  `json:"fromTs"`
	Note   string `json:"note"`
}

const tireDDL = `CREATE TABLE IF NOT EXISTS tire_sets (
 id INTEGER PRIMARY KEY, name TEXT NOT NULL, kind TEXT NOT NULL DEFAULT 'other', from_ts INTEGER NOT NULL, note TEXT NOT NULL DEFAULT '')`

// TireKinds are the kinds a set can be.
var TireKinds = []string{"winter", "summer", "allseason", "other"}

// ValidTireKind says whether kind is one of TireKinds.
func ValidTireKind(kind string) bool {
	for _, k := range TireKinds {
		if k == kind {
			return true
		}
	}
	return false
}

// TireSets lists the sets, oldest first.
func (s *Store) TireSets() ([]TireSet, error) {
	if _, err := s.DB.Exec(tireDDL); err != nil {
		return nil, err
	}
	rows, err := s.DB.Query(`SELECT id,name,kind,from_ts,note FROM tire_sets ORDER BY from_ts, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []TireSet{}
	for rows.Next() {
		var t TireSet
		if err := rows.Scan(&t.ID, &t.Name, &t.Kind, &t.FromTS, &t.Note); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// AddTireSet records a set going on and returns its id.
func (s *Store) AddTireSet(t TireSet) (int64, error) {
	if _, err := s.DB.Exec(tireDDL); err != nil {
		return 0, err
	}
	res, err := s.DB.Exec(`INSERT INTO tire_sets(name,kind,from_ts,note) VALUES(?,?,?,?)`, strings.TrimSpace(t.Name), t.Kind, t.FromTS, strings.TrimSpace(t.Note))
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateTireSet changes a set's name, kind, date and note.
func (s *Store) UpdateTireSet(t TireSet) error {
	if _, err := s.DB.Exec(tireDDL); err != nil {
		return err
	}
	_, err := s.DB.Exec(`UPDATE tire_sets SET name=?,kind=?,from_ts=?,note=? WHERE id=?`, strings.TrimSpace(t.Name), t.Kind, t.FromTS, strings.TrimSpace(t.Note), t.ID)
	return err
}

// DeleteTireSet removes a set; the trips it covered fall back to the set before it.
func (s *Store) DeleteTireSet(id int64) error {
	if _, err := s.DB.Exec(tireDDL); err != nil {
		return err
	}
	_, err := s.DB.Exec(`DELETE FROM tire_sets WHERE id=?`, id)
	return err
}

// TireAt is the set the car was wearing at ts: the latest one that went on at or before it, or nil.
func TireAt(sets []TireSet, ts int64) *TireSet {
	var best *TireSet
	for i := range sets {
		if sets[i].FromTS <= ts && (best == nil || sets[i].FromTS >= best.FromTS) {
			best = &sets[i]
		}
	}
	return best
}
