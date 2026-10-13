package store

import "database/sql"

type Charge struct {
	ID       int64    `json:"id"`
	StartTS  int64    `json:"startTs"`
	EndTS    *int64   `json:"endTs"`
	Lat      *float64 `json:"lat"`
	Lon      *float64 `json:"lon"`
	StartSoC *float64 `json:"startSoc"`
	EndSoC   *float64 `json:"endSoc"`
	PeakKw   float64  `json:"peakKw"`
	Open     bool     `json:"open"`
	StartKwh *float64 `json:"startKwh"` // energy left in the battery at the start and end, kWh
	EndKwh   *float64 `json:"endKwh"`
	Kind     string   `json:"kind"` // "home", "public", or "" when CarTrak decides from the place
}

const chargeCols = `id,start_ts,end_ts,lat,lon,start_soc,end_soc,peak_kw,open,start_kwh,end_kwh,kind`

func scanCharge(r interface{ Scan(...any) error }) (*Charge, error) {
	var c Charge
	err := r.Scan(&c.ID, &c.StartTS, &c.EndTS, &c.Lat, &c.Lon, &c.StartSoC, &c.EndSoC, &c.PeakKw, &c.Open, &c.StartKwh, &c.EndKwh, &c.Kind)
	return &c, err
}

func (s *Store) OpenCharge() (*Charge, error) {
	c, err := scanCharge(s.DB.QueryRow(`SELECT ` + chargeCols + ` FROM charges WHERE open=1 ORDER BY start_ts DESC LIMIT 1`))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return c, err
}

func (s *Store) CreateCharge(c *Charge) error {
	r, err := s.DB.Exec(`INSERT INTO charges(start_ts,lat,lon,start_soc,end_soc,peak_kw,start_kwh,end_kwh,open) VALUES(?,?,?,?,?,?,?,?,1)`,
		c.StartTS, c.Lat, c.Lon, c.StartSoC, c.StartSoC, c.PeakKw, c.StartKwh, c.StartKwh)
	if err != nil {
		return err
	}
	c.ID, _ = r.LastInsertId()
	c.Open = true
	return nil
}

func (s *Store) UpdateCharge(c *Charge) error {
	_, err := s.DB.Exec(`UPDATE charges SET end_ts=?,end_soc=?,end_kwh=?,peak_kw=?,open=? WHERE id=?`,
		c.EndTS, c.EndSoC, c.EndKwh, c.PeakKw, c.Open, c.ID)
	return err
}

func (s *Store) ListCharges(limit int) ([]*Charge, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	rows, err := s.DB.Query(`SELECT `+chargeCols+` FROM charges ORDER BY start_ts DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Charge{}
	for rows.Next() {
		c, err := scanCharge(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) GetCharge(id int64) (*Charge, error) {
	c, err := scanCharge(s.DB.QueryRow(`SELECT `+chargeCols+` FROM charges WHERE id=?`, id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return c, err
}

// SetChargeKind records the owner's choice: "home", "public", or "" to let CarTrak decide from the place.
func (s *Store) SetChargeKind(id int64, kind string) error {
	_, err := s.DB.Exec(`UPDATE charges SET kind=? WHERE id=?`, kind, id)
	return err
}
