package store

import (
	"database/sql"
	"strings"
	"sync"
	"time"
)

// A service visit: the car at a dealership, tire shop or repair shop, found from a long stop there (Status "pending" until the
// owner says what was done, or "dismissed" when it was not service) or logged by hand. Logged visits are the service history.

type ServiceVisit struct {
	ID        int64    `json:"id"`
	PlaceID   int64    `json:"placeId"`
	PlaceName string   `json:"placeName"`
	Kind      string   `json:"kind"` // the place's service kind when it was found: dealer, tire, repair or ""
	Start     int64    `json:"start"`
	End       int64    `json:"end"`
	OdoKm     *float64 `json:"odoKm"`
	Status    string   `json:"status"` // pending, logged or dismissed
	Items     []string `json:"items"`  // what was done, such as "Tire rotation"
	Notes     string   `json:"notes"`
	Cost      *float64 `json:"cost"`
}

const serviceDDL = `CREATE TABLE IF NOT EXISTS service_visits (
  id INTEGER PRIMARY KEY, place_id INTEGER NOT NULL DEFAULT 0, place_name TEXT NOT NULL DEFAULT '', kind TEXT NOT NULL DEFAULT '',
  start_ts INTEGER NOT NULL, end_ts INTEGER NOT NULL, odo_km REAL, status TEXT NOT NULL, items TEXT NOT NULL DEFAULT '',
  notes TEXT NOT NULL DEFAULT '', cost REAL);
CREATE INDEX IF NOT EXISTS service_visits_start ON service_visits(start_ts);`

var serviceTables sync.Map // *Store -> struct{}: the table is made once per database

func (s *Store) ensureService() error {
	if _, done := serviceTables.Load(s); done {
		return nil
	}
	if _, err := s.DB.Exec(serviceDDL); err != nil {
		return err
	}
	serviceTables.Store(s, struct{}{})
	return nil
}

const serviceCols = `id,place_id,place_name,kind,start_ts,end_ts,odo_km,status,items,notes,cost`

func scanVisit(row interface{ Scan(...any) error }) (*ServiceVisit, error) {
	var v ServiceVisit
	var items string
	if err := row.Scan(&v.ID, &v.PlaceID, &v.PlaceName, &v.Kind, &v.Start, &v.End, &v.OdoKm, &v.Status, &items, &v.Notes, &v.Cost); err != nil {
		return nil, err
	}
	v.Items = []string{}
	for _, it := range strings.Split(items, "\n") {
		if it = strings.TrimSpace(it); it != "" {
			v.Items = append(v.Items, it)
		}
	}
	return &v, nil
}

// SaveVisit adds a visit (ID 0) or updates one.
func (s *Store) SaveVisit(v *ServiceVisit) error {
	if err := s.ensureService(); err != nil {
		return err
	}
	items := strings.Join(v.Items, "\n")
	if v.ID == 0 {
		r, err := s.DB.Exec(`INSERT INTO service_visits(place_id,place_name,kind,start_ts,end_ts,odo_km,status,items,notes,cost) VALUES(?,?,?,?,?,?,?,?,?,?)`,
			v.PlaceID, v.PlaceName, v.Kind, v.Start, v.End, v.OdoKm, v.Status, items, v.Notes, v.Cost)
		if err == nil {
			v.ID, _ = r.LastInsertId()
		}
		return err
	}
	_, err := s.DB.Exec(`UPDATE service_visits SET place_name=?,start_ts=?,end_ts=?,odo_km=?,status=?,items=?,notes=?,cost=? WHERE id=?`,
		v.PlaceName, v.Start, v.End, v.OdoKm, v.Status, items, v.Notes, v.Cost, v.ID)
	return err
}

func (s *Store) GetVisit(id int64) (*ServiceVisit, error) {
	if err := s.ensureService(); err != nil {
		return nil, err
	}
	v, err := scanVisit(s.DB.QueryRow(`SELECT `+serviceCols+` FROM service_visits WHERE id=?`, id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return v, err
}

// Visits lists visits that began in from..to (to 0 is now), newest first; status "" means any.
func (s *Store) Visits(from, to int64, status string) ([]*ServiceVisit, error) {
	if err := s.ensureService(); err != nil {
		return nil, err
	}
	if to == 0 {
		to = time.Now().Unix() + 86400
	}
	rows, err := s.DB.Query(`SELECT `+serviceCols+` FROM service_visits WHERE start_ts>=? AND start_ts<? AND (?='' OR status=?) ORDER BY start_ts DESC`, from, to, status, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*ServiceVisit{}
	for rows.Next() {
		v, err := scanVisit(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// HasVisitAt says whether a visit at this place starting at this moment is already known, so a stop is asked about once.
func (s *Store) HasVisitAt(placeID, start int64) bool {
	if s.ensureService() != nil {
		return true
	}
	var n int
	_ = s.DB.QueryRow(`SELECT COUNT(*) FROM service_visits WHERE place_id=? AND start_ts=?`, placeID, start).Scan(&n)
	return n > 0
}

func (s *Store) DeleteVisit(id int64) error {
	if err := s.ensureService(); err != nil {
		return err
	}
	_, err := s.DB.Exec(`DELETE FROM service_visits WHERE id=?`, id)
	return err
}
