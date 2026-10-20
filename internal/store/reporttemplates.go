package store

import (
	"errors"
	"strings"
	"time"
)

// ReportTemplate is a saved custom report: a name and its layout, kept as JSON so the layout can grow without a new table.
type ReportTemplate struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Config  string `json:"config"`
	Created int64  `json:"created"`
}

const reportTemplatesSchema = `CREATE TABLE IF NOT EXISTS report_templates (
  id INTEGER PRIMARY KEY, name TEXT NOT NULL, config TEXT NOT NULL, created INTEGER NOT NULL)`

// MaxReportTemplates is how many custom reports a car can keep.
const MaxReportTemplates = 20

func (s *Store) ListReportTemplates() ([]ReportTemplate, error) {
	if _, err := s.DB.Exec(reportTemplatesSchema); err != nil {
		return nil, err
	}
	rows, err := s.DB.Query(`SELECT id,name,config,created FROM report_templates ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ReportTemplate{}
	for rows.Next() {
		var t ReportTemplate
		if err := rows.Scan(&t.ID, &t.Name, &t.Config, &t.Created); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) GetReportTemplate(id int64) (*ReportTemplate, error) {
	ts, err := s.ListReportTemplates()
	if err != nil {
		return nil, err
	}
	for i := range ts {
		if ts[i].ID == id {
			return &ts[i], nil
		}
	}
	return nil, nil
}

// SaveReportTemplate adds a custom report, or replaces the one with t.ID.
func (s *Store) SaveReportTemplate(t *ReportTemplate) error {
	t.Name = strings.Join(strings.Fields(t.Name), " ")
	if t.Name == "" || len([]rune(t.Name)) > 40 {
		return errors.New("give the report a short name (up to 40 characters)")
	}
	if _, err := s.DB.Exec(reportTemplatesSchema); err != nil {
		return err
	}
	if t.ID == 0 {
		var n int
		_ = s.DB.QueryRow(`SELECT COUNT(*) FROM report_templates`).Scan(&n)
		if n >= MaxReportTemplates {
			return errors.New("that is the most custom reports a car can keep")
		}
		r, err := s.DB.Exec(`INSERT INTO report_templates(name,config,created) VALUES(?,?,?)`, t.Name, t.Config, time.Now().Unix())
		if err != nil {
			return err
		}
		t.ID, _ = r.LastInsertId()
		return nil
	}
	r, err := s.DB.Exec(`UPDATE report_templates SET name=?,config=? WHERE id=?`, t.Name, t.Config, t.ID)
	if err != nil {
		return err
	}
	if n, _ := r.RowsAffected(); n == 0 {
		return errors.New("no such report")
	}
	return nil
}

func (s *Store) DeleteReportTemplate(id int64) error {
	if _, err := s.DB.Exec(reportTemplatesSchema); err != nil {
		return err
	}
	_, err := s.DB.Exec(`DELETE FROM report_templates WHERE id=?`, id)
	return err
}

// TripsBetween returns every trip that started in [from, to), oldest first, with no limit on how many. Reports need all of them.
func (s *Store) TripsBetween(from, to int64) ([]*Trip, error) {
	rows, err := s.DB.Query(`SELECT `+tripCols+` FROM trips WHERE start_ts>=? AND start_ts<? ORDER BY start_ts`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Trip{}
	for rows.Next() {
		t, err := scanTrip(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
