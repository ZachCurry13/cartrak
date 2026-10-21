package store

import (
	"database/sql"

	"github.com/ZachCurry13/cartrak/internal/fuel"
)

// Fill-ups of a gas or hybrid car: found from the fuel gauge jumping up, completed by hand with the gallons and the price.

const refuelCols = `id,ts,start_pct,end_pct,odo_km,gallons,price_per_gal,status,source`

func scanRefuel(r interface{ Scan(...any) error }) (*fuel.Fill, error) {
	var f fuel.Fill
	err := r.Scan(&f.ID, &f.TS, &f.StartPct, &f.EndPct, &f.OdoKm, &f.Gallons, &f.PricePerGal, &f.Status, &f.Source)
	return &f, err
}

// SaveRefuel adds a fill-up (ID 0) or updates one.
func (s *Store) SaveRefuel(f *fuel.Fill) error {
	if f.Status == "" {
		f.Status = "pending"
	}
	if f.Source == "" {
		f.Source = "auto"
	}
	if f.ID == 0 {
		r, err := s.DB.Exec(`INSERT INTO refuels(ts,start_pct,end_pct,odo_km,gallons,price_per_gal,status,source) VALUES(?,?,?,?,?,?,?,?)`,
			f.TS, f.StartPct, f.EndPct, f.OdoKm, f.Gallons, f.PricePerGal, f.Status, f.Source)
		if err == nil {
			f.ID, _ = r.LastInsertId()
		}
		return err
	}
	_, err := s.DB.Exec(`UPDATE refuels SET ts=?,start_pct=?,end_pct=?,odo_km=?,gallons=?,price_per_gal=?,status=? WHERE id=?`,
		f.TS, f.StartPct, f.EndPct, f.OdoKm, f.Gallons, f.PricePerGal, f.Status, f.ID)
	return err
}

func (s *Store) GetRefuel(id int64) (*fuel.Fill, error) {
	f, err := scanRefuel(s.DB.QueryRow(`SELECT `+refuelCols+` FROM refuels WHERE id=?`, id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return f, err
}

func (s *Store) DeleteRefuel(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM refuels WHERE id=?`, id)
	return err
}

// ListRefuels is every fill-up that was not dismissed, oldest first.
func (s *Store) ListRefuels() ([]fuel.Fill, error) {
	rows, err := s.DB.Query(`SELECT ` + refuelCols + ` FROM refuels WHERE status!='dismissed' ORDER BY ts`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []fuel.Fill{}
	for rows.Next() {
		f, err := scanRefuel(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *f)
	}
	return out, rows.Err()
}

// LastRefuel is the newest fill-up of any status, or nil.
func (s *Store) LastRefuel() (*fuel.Fill, error) {
	f, err := scanRefuel(s.DB.QueryRow(`SELECT ` + refuelCols + ` FROM refuels ORDER BY ts DESC LIMIT 1`))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return f, err
}

// PendingRefuels counts fill-ups found but not yet completed with the gallons and the price.
func (s *Store) PendingRefuels() int {
	var n int
	_ = s.DB.QueryRow(`SELECT COUNT(*) FROM refuels WHERE status='pending'`).Scan(&n)
	return n
}

// FuelPoint is the tank level at a moment.
type FuelPoint struct {
	TS  int64
	Pct float64
}

// FuelBetween is the tank levels the car reported from one time to another, oldest first.
func (s *Store) FuelBetween(from, to int64) ([]FuelPoint, error) {
	rows, err := s.DB.Query(`SELECT ts,fuel_pct FROM trends WHERE ts>=? AND ts<=? AND fuel_pct IS NOT NULL ORDER BY ts`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []FuelPoint
	for rows.Next() {
		var p FuelPoint
		if err := rows.Scan(&p.TS, &p.Pct); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
