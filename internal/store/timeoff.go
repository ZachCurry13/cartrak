package store

import "fmt"

// TimeOff is a stretch of days when every trip gets the same category whatever the rules say: Personal (a vacation), Work (a
// work trip), Medical, or any other category.
type TimeOff struct {
	ID    int64  `json:"id"`
	From  string `json:"from"` // first day, YYYY-MM-DD in the car's time zone
	To    string `json:"to"`   // last day, inclusive
	Label string `json:"label"`
	Kind  string `json:"kind"` // "vacation" (trips are Personal), or the key of the category the trips get
	// BaseID is the named place chosen as where you are staying, or nil to let CarTrak find it from where the car spends the nights.
	BaseID *int64 `json:"baseId"`
}

// KindVacation is the original mode: trips are Personal.
const KindVacation = "vacation"

const timeOffSchema = `CREATE TABLE IF NOT EXISTS time_off (
  id INTEGER PRIMARY KEY, from_day TEXT NOT NULL, to_day TEXT NOT NULL, label TEXT NOT NULL DEFAULT '')`

func (s *Store) ensureTimeOff() error {
	if _, err := s.DB.Exec(timeOffSchema); err != nil {
		return err
	}
	if has, err := hasColumn(s.DB, "time_off", "kind"); err == nil && !has { // stretches made before work trips existed are vacations
		if _, err = s.DB.Exec(`ALTER TABLE time_off ADD COLUMN kind TEXT NOT NULL DEFAULT 'vacation'`); err != nil {
			return err
		}
	}
	if has, err := hasColumn(s.DB, "time_off", "base_id"); err == nil && !has {
		_, err = s.DB.Exec(`ALTER TABLE time_off ADD COLUMN base_id INTEGER`)
		return err
	}
	return nil
}

// ListTimeOff returns every stretch, newest first.
func (s *Store) ListTimeOff() ([]TimeOff, error) {
	if err := s.ensureTimeOff(); err != nil {
		return nil, err
	}
	rows, err := s.DB.Query(`SELECT id,from_day,to_day,label,kind,base_id FROM time_off ORDER BY from_day DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []TimeOff{}
	for rows.Next() {
		var t TimeOff
		if err := rows.Scan(&t.ID, &t.From, &t.To, &t.Label, &t.Kind, &t.BaseID); err != nil {
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
	if t.Kind == "" {
		t.Kind = KindVacation
	}
	if t.Kind != KindVacation && !s.CategoryExists(t.Kind) {
		return fmt.Errorf("choose a vacation or one of your categories")
	}
	r, err := s.DB.Exec(`INSERT INTO time_off(from_day,to_day,label,kind) VALUES(?,?,?,?)`, t.From, t.To, t.Label, t.Kind)
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

// TimeOffKind says what a day (YYYY-MM-DD) is: the kind of the newest stretch that covers it, or "" when none does. The newest
// wins, so a work trip added in the middle of a vacation week takes its days.
func (s *Store) TimeOffKind(day string) string {
	if err := s.ensureTimeOff(); err != nil {
		return ""
	}
	var kind string
	_ = s.DB.QueryRow(`SELECT kind FROM time_off WHERE from_day<=? AND to_day>=? ORDER BY id DESC LIMIT 1`, day, day).Scan(&kind)
	return kind
}

// StretchOn is the newest stretch that covers a day (YYYY-MM-DD), or nil.
func (s *Store) StretchOn(day string) *TimeOff {
	if err := s.ensureTimeOff(); err != nil {
		return nil
	}
	var t TimeOff
	err := s.DB.QueryRow(`SELECT id,from_day,to_day,label,kind,base_id FROM time_off WHERE from_day<=? AND to_day>=? ORDER BY id DESC LIMIT 1`, day, day).
		Scan(&t.ID, &t.From, &t.To, &t.Label, &t.Kind, &t.BaseID)
	if err != nil {
		return nil
	}
	return &t
}

// SetTimeOffBase chooses the named place where you are staying for a stretch, or nil to find it from the car's nights.
func (s *Store) SetTimeOffBase(id int64, base *int64) error {
	if err := s.ensureTimeOff(); err != nil {
		return err
	}
	_, err := s.DB.Exec(`UPDATE time_off SET base_id=? WHERE id=?`, base, id)
	return err
}

// PositionAt is where the car last reported being at or before a moment, looking back at most maxAge seconds.
func (s *Store) PositionAt(ts, maxAge int64) (lat, lon float64, ok bool) {
	err := s.DB.QueryRow(`SELECT lat,lon FROM samples WHERE ts<=? AND ts>? AND lat IS NOT NULL AND lon IS NOT NULL ORDER BY ts DESC LIMIT 1`, ts, ts-maxAge).Scan(&lat, &lon)
	return lat, lon, err == nil
}

// OnTimeOff says whether a day (YYYY-MM-DD) falls inside any stretch.
func (s *Store) OnTimeOff(day string) bool { return s.TimeOffKind(day) != "" }
