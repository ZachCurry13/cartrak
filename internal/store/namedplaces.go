package store

import "database/sql"

// NamedPlace is somewhere the owner has named, such as Home or Office, with how close counts as being there.
type NamedPlace struct {
	ID      int64   `json:"id"`
	Name    string  `json:"name"`
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
	RadiusM float64 `json:"radiusM"`
}

func (s *Store) ListNamedPlaces() ([]*NamedPlace, error) {
	rows, err := s.DB.Query(`SELECT id,name,lat,lon,radius_m FROM named_places ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*NamedPlace{}
	for rows.Next() {
		var p NamedPlace
		if err := rows.Scan(&p.ID, &p.Name, &p.Lat, &p.Lon, &p.RadiusM); err != nil {
			return nil, err
		}
		out = append(out, &p)
	}
	return out, rows.Err()
}

func (s *Store) GetNamedPlace(id int64) (*NamedPlace, error) {
	var p NamedPlace
	err := s.DB.QueryRow(`SELECT id,name,lat,lon,radius_m FROM named_places WHERE id=?`, id).Scan(&p.ID, &p.Name, &p.Lat, &p.Lon, &p.RadiusM)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &p, err
}

func (s *Store) SaveNamedPlace(p *NamedPlace) error {
	if p.ID == 0 {
		r, err := s.DB.Exec(`INSERT INTO named_places(name,lat,lon,radius_m) VALUES(?,?,?,?)`, p.Name, p.Lat, p.Lon, p.RadiusM)
		if err != nil {
			return err
		}
		p.ID, _ = r.LastInsertId()
		return nil
	}
	_, err := s.DB.Exec(`UPDATE named_places SET name=?,lat=?,lon=?,radius_m=? WHERE id=?`, p.Name, p.Lat, p.Lon, p.RadiusM, p.ID)
	return err
}

func (s *Store) DeleteNamedPlace(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM named_places WHERE id=?`, id)
	return err
}
