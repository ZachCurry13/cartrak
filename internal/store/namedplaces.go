package store

import "database/sql"

// NamedPlace is somewhere the owner has named, such as Home or Office, with how close counts as being there.
type NamedPlace struct {
	ID      int64   `json:"id"`
	Name    string  `json:"name"`
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
	RadiusM float64 `json:"radiusM"`
	// ChargePrice is what charging at this place costs per kWh, when it is not the usual home or public price (a cabin, a
	// relative's house, a workplace charger, free charging at 0). Nil means the usual price applies.
	ChargePrice *float64 `json:"chargePrice"`
	// ChargeHome says charging here counts as home charging: the home price, and home in the statistics.
	ChargeHome bool `json:"chargeHome"`
}

func (s *Store) ListNamedPlaces() ([]*NamedPlace, error) {
	rows, err := s.DB.Query(`SELECT id,name,lat,lon,radius_m,charge_price,charge_home FROM named_places ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*NamedPlace{}
	for rows.Next() {
		var p NamedPlace
		if err := rows.Scan(&p.ID, &p.Name, &p.Lat, &p.Lon, &p.RadiusM, &p.ChargePrice, &p.ChargeHome); err != nil {
			return nil, err
		}
		out = append(out, &p)
	}
	return out, rows.Err()
}

func (s *Store) GetNamedPlace(id int64) (*NamedPlace, error) {
	var p NamedPlace
	err := s.DB.QueryRow(`SELECT id,name,lat,lon,radius_m,charge_price,charge_home FROM named_places WHERE id=?`, id).Scan(&p.ID, &p.Name, &p.Lat, &p.Lon, &p.RadiusM, &p.ChargePrice, &p.ChargeHome)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &p, err
}

func (s *Store) SaveNamedPlace(p *NamedPlace) error {
	if p.ID == 0 {
		r, err := s.DB.Exec(`INSERT INTO named_places(name,lat,lon,radius_m,charge_price,charge_home) VALUES(?,?,?,?,?,?)`, p.Name, p.Lat, p.Lon, p.RadiusM, p.ChargePrice, p.ChargeHome)
		if err != nil {
			return err
		}
		p.ID, _ = r.LastInsertId()
		return nil
	}
	_, err := s.DB.Exec(`UPDATE named_places SET name=?,lat=?,lon=?,radius_m=?,charge_price=?,charge_home=? WHERE id=?`, p.Name, p.Lat, p.Lon, p.RadiusM, p.ChargePrice, p.ChargeHome, p.ID)
	return err
}

func (s *Store) DeleteNamedPlace(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM named_places WHERE id=?`, id)
	return err
}
