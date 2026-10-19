package store

import (
	"database/sql"
)

// ReplayRow is one reading with the fields the trip and charge detector reads. No raw detail, so a few days of
// them read quickly.
type ReplayRow struct {
	TS          int64
	OdometerKm  *float64
	SoC         *float64
	Lat, Lon    *float64
	SpeedKph    *float64
	Ignition    string
	ChargeState string
	ChargeKw    *float64
	EnergyKwh   *float64
}

// ReplayRows returns the readings from `from` to `to`, oldest first.
func (s *Store) ReplayRows(from, to int64) ([]ReplayRow, error) {
	rows, err := s.DB.Query(`SELECT ts,odometer_km,soc_pct,lat,lon,speed_kph,ignition,charge_state,charge_kw,energy_kwh
		FROM samples WHERE ts BETWEEN ? AND ? ORDER BY ts`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ReplayRow
	for rows.Next() {
		var r ReplayRow
		var ign, cs sql.NullString
		if err := rows.Scan(&r.TS, &r.OdometerKm, &r.SoC, &r.Lat, &r.Lon, &r.SpeedKph, &ign, &cs, &r.ChargeKw, &r.EnergyKwh); err != nil {
			return nil, err
		}
		r.Ignition, r.ChargeState = ign.String, cs.String
		out = append(out, r)
	}
	return out, rows.Err()
}

// TripsTouching returns the trips that overlap [from, to] (or are still open), oldest first.
func (s *Store) TripsTouching(from, to int64) ([]*Trip, error) {
	rows, err := s.DB.Query(`SELECT `+tripCols+` FROM trips WHERE start_ts<=? AND (COALESCE(end_ts,start_ts)>=? OR open=1) ORDER BY start_ts`, to, from)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Trip
	for rows.Next() {
		t, err := scanTrip(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// ChargesTouching returns the charging sessions that overlap [from, to] (or are still open), oldest first.
func (s *Store) ChargesTouching(from, to int64) ([]*Charge, error) {
	rows, err := s.DB.Query(`SELECT `+chargeCols+` FROM charges WHERE start_ts<=? AND (COALESCE(end_ts,start_ts)>=? OR open=1) ORDER BY start_ts`, to, from)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Charge
	for rows.Next() {
		c, err := scanCharge(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ReplaceRange swaps trips and charging sessions in one step: the old ones (and any public link made from an old
// trip) go and the new ones are added, or nothing changes at all.
func (s *Store) ReplaceRange(oldTrips []int64, oldCharges []int64, newTrips []*Trip, newCharges []*Charge) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, id := range oldTrips {
		if _, err := tx.Exec(`DELETE FROM shares WHERE kind='trip' AND target_id=?`, id); err != nil {
			return err
		}
		if _, err := tx.Exec(`DELETE FROM trips WHERE id=?`, id); err != nil {
			return err
		}
	}
	for _, id := range oldCharges {
		if _, err := tx.Exec(`DELETE FROM charges WHERE id=?`, id); err != nil {
			return err
		}
	}
	for _, t := range newTrips {
		r, err := tx.Exec(`INSERT INTO trips(start_ts,end_ts,start_lat,start_lon,end_lat,end_lon,start_odo,end_odo,distance_km,start_soc,end_soc,
			max_speed_kph,category,category_source,rule_id,notes,open,tz,start_kwh,end_kwh) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,0,?,?,?)`,
			t.StartTS, t.EndTS, t.StartLat, t.StartLon, t.EndLat, t.EndLon, t.StartOdo, t.EndOdo, t.DistanceKm, t.StartSoC, t.EndSoC,
			t.MaxSpeedKph, t.Category, t.CategorySource, t.RuleID, t.Notes, t.TZ, t.StartKwh, t.EndKwh)
		if err != nil {
			return err
		}
		t.ID, _ = r.LastInsertId()
	}
	for _, c := range newCharges {
		r, err := tx.Exec(`INSERT INTO charges(start_ts,end_ts,lat,lon,start_soc,end_soc,peak_kw,open,start_kwh,end_kwh,kind,charger_kwh)
			VALUES(?,?,?,?,?,?,?,0,?,?,?,?)`, c.StartTS, c.EndTS, c.Lat, c.Lon, c.StartSoC, c.EndSoC, c.PeakKw, c.StartKwh, c.EndKwh, c.Kind, c.ChargerKwh)
		if err != nil {
			return err
		}
		c.ID, _ = r.LastInsertId()
	}
	return tx.Commit()
}
