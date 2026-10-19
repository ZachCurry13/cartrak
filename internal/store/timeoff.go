package store

import "fmt"

// TimeOff is a stretch of days (vacation) when trips count as personal whatever the rules say.
type TimeOff struct {
	ID    int64  `json:"id"`
	From  string `json:"from"` // first day, YYYY-MM-DD in the car's time zone
	To    string `json:"to"`   // last day, inclusive
	Label string `json:"label"`
}

const timeOffSchema = `CREATE TABLE IF NOT EXISTS time_off (
  id INTEGER PRIMARY KEY, from_day TEXT NOT NULL, to_day TEXT NOT NULL, label TEXT NOT NULL DEFAULT '')`

func (s *Store) ensureTimeOff() error {
	_, err := s.DB.Exec(timeOffSchema)
	return err
}

// ListTimeOff returns every stretch, newest first.
func (s *Store) ListTimeOff() ([]TimeOff, error) {
	if err := s.ensureTimeOff(); err != nil {
		return nil, err
	}
	rows, err := s.DB.Query(`SELECT id,from_day,to_day,label FROM time_off ORDER BY from_day DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []TimeOff{}
	for rows.Next() {
		var t TimeOff
		if err := rows.Scan(&t.ID, &t.From, &t.To, &t.Label); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// AddTimeOff saves a stretch. The days must be YYYY-MM-DD and the end not before the start.
func (s *Store) AddTimeOff(t *TimeOff) error {
	if err := s.ensureTimeOff(); err != nil {
		return err
	}
	if len(t.From) != 10 || len(t.To) != 10 || t.To < t.From {
		return fmt.Errorf("the end day must not be before the start day")
	}
	r, err := s.DB.Exec(`INSERT INTO time_off(from_day,to_day,label) VALUES(?,?,?)`, t.From, t.To, t.Label)
	if err != nil {
		return err
	}
	t.ID, _ = r.LastInsertId()
	return nil
}

func (s *Store) DeleteTimeOff(id int64) error {
	if err := s.ensureTimeOff(); err != nil {
		return err
	}
	_, err := s.DB.Exec(`DELETE FROM time_off WHERE id=?`, id)
	return err
}

// OnTimeOff says whether a day (YYYY-MM-DD) falls inside any stretch.
func (s *Store) OnTimeOff(day string) bool {
	if err := s.ensureTimeOff(); err != nil {
		return false
	}
	var n int
	_ = s.DB.QueryRow(`SELECT COUNT(*) FROM time_off WHERE from_day<=? AND to_day>=?`, day, day).Scan(&n)
	return n > 0
}
