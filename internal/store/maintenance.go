package store

import (
	"database/sql"
	"time"
)

// Maintenance is a service reminder: something to do every so many kilometres, every so many days, or both.
type Maintenance struct {
	ID         int64    `json:"id"`
	Name       string   `json:"name"`
	EveryKm    *float64 `json:"everyKm"`
	EveryDays  *int     `json:"everyDays"`
	LastOdoKm  *float64 `json:"lastOdoKm"` // odometer when it was last done
	LastTS     int64    `json:"lastTs"`    // when it was last done
	NotifiedTS int64    `json:"notifiedTs"`
}

const maintCols = `id,name,every_km,every_days,last_odo_km,last_ts,notified_ts`

func scanMaint(r interface{ Scan(...any) error }) (*Maintenance, error) {
	var m Maintenance
	err := r.Scan(&m.ID, &m.Name, &m.EveryKm, &m.EveryDays, &m.LastOdoKm, &m.LastTS, &m.NotifiedTS)
	return &m, err
}

func (s *Store) ListMaintenance() ([]*Maintenance, error) {
	rows, err := s.DB.Query(`SELECT ` + maintCols + ` FROM maintenance ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Maintenance{}
	for rows.Next() {
		m, err := scanMaint(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) GetMaintenance(id int64) (*Maintenance, error) {
	m, err := scanMaint(s.DB.QueryRow(`SELECT `+maintCols+` FROM maintenance WHERE id=?`, id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return m, err
}

func (s *Store) SaveMaintenance(m *Maintenance) error {
	if m.ID == 0 {
		r, err := s.DB.Exec(`INSERT INTO maintenance(name,every_km,every_days,last_odo_km,last_ts,notified_ts) VALUES(?,?,?,?,?,0)`,
			m.Name, m.EveryKm, m.EveryDays, m.LastOdoKm, m.LastTS)
		if err != nil {
			return err
		}
		m.ID, _ = r.LastInsertId()
		return nil
	}
	_, err := s.DB.Exec(`UPDATE maintenance SET name=?,every_km=?,every_days=?,last_odo_km=?,last_ts=?,notified_ts=? WHERE id=?`,
		m.Name, m.EveryKm, m.EveryDays, m.LastOdoKm, m.LastTS, m.NotifiedTS, m.ID)
	return err
}

// MarkMaintenanceDone records that the work was done now, at this odometer, and re-arms the reminder.
func (s *Store) MarkMaintenanceDone(id int64, odoKm *float64) error {
	_, err := s.DB.Exec(`UPDATE maintenance SET last_odo_km=?,last_ts=?,notified_ts=0 WHERE id=?`, odoKm, time.Now().Unix(), id)
	return err
}

func (s *Store) SetMaintenanceNotified(id int64) error {
	_, err := s.DB.Exec(`UPDATE maintenance SET notified_ts=? WHERE id=?`, time.Now().Unix(), id)
	return err
}

func (s *Store) DeleteMaintenance(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM maintenance WHERE id=?`, id)
	return err
}
