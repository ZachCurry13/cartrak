package store

import "database/sql"

// Rule auto-classifies a trip. Days is a bitmask (Sun=1, Mon=2 ... Sat=64);
// StartMin/EndMin are minutes after local midnight. An optional geofence
// requires the trip to start within GeoRadius meters of the point.
type Rule struct {
	ID        int64    `json:"id"`
	Name      string   `json:"name"`
	Category  string   `json:"category"`
	Priority  int      `json:"priority"`
	Enabled   bool     `json:"enabled"`
	Days      int      `json:"days"`
	StartMin  int      `json:"startMin"`
	EndMin    int      `json:"endMin"`
	GeoLat    *float64 `json:"geoLat"`
	GeoLon    *float64 `json:"geoLon"`
	GeoRadius *float64 `json:"geoRadius"`
}

func (s *Store) ListRules() ([]Rule, error) {
	rows, err := s.DB.Query(`SELECT id,name,category,priority,enabled,days,start_min,end_min,
		geo_lat,geo_lon,geo_radius_m FROM rules ORDER BY priority DESC, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Rule{}
	for rows.Next() {
		var r Rule
		if err := rows.Scan(&r.ID, &r.Name, &r.Category, &r.Priority, &r.Enabled, &r.Days,
			&r.StartMin, &r.EndMin, &r.GeoLat, &r.GeoLon, &r.GeoRadius); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// SaveRule inserts (ID==0) or updates a rule and returns its id.
func (s *Store) SaveRule(r Rule) (int64, error) {
	if r.ID == 0 {
		res, err := s.DB.Exec(`INSERT INTO rules(name,category,priority,enabled,days,start_min,end_min,
			geo_lat,geo_lon,geo_radius_m) VALUES(?,?,?,?,?,?,?,?,?,?)`,
			r.Name, r.Category, r.Priority, r.Enabled, r.Days, r.StartMin, r.EndMin, r.GeoLat, r.GeoLon, r.GeoRadius)
		if err != nil {
			return 0, err
		}
		return res.LastInsertId()
	}
	_, err := s.DB.Exec(`UPDATE rules SET name=?,category=?,priority=?,enabled=?,days=?,start_min=?,
		end_min=?,geo_lat=?,geo_lon=?,geo_radius_m=? WHERE id=?`,
		r.Name, r.Category, r.Priority, r.Enabled, r.Days, r.StartMin, r.EndMin, r.GeoLat, r.GeoLon, r.GeoRadius, r.ID)
	return r.ID, err
}

func (s *Store) DeleteRule(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM rules WHERE id=?`, id)
	return err
}

var _ = sql.ErrNoRows
