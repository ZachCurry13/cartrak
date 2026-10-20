package store

import "database/sql"

// DebugRow is one reading with only the few fields the debug report lists. No position and none of the large raw
// fields, so reading three days of them is fast and carries no location.
type DebugRow struct {
	TS          int64
	Ignition    string
	ChargeState string
	Plug        string
	ChargeKw    *float64
	SpeedKph    *float64
	SoC         *float64
	OdometerKm  *float64
	Locked      *bool
	EnergyKwh   *float64 // energy left in the battery
	PackKw      *float64 // power worked out from the pack's current and voltage when the car reported none (fast chargers)
}

// DebugRows returns the readings from `from` to `to`, oldest first.
func (s *Store) DebugRows(from, to int64) ([]DebugRow, error) {
	rows, err := s.DB.Query(`SELECT ts,ignition,charge_state,plug,charge_kw,speed_kph,soc_pct,odometer_km,locked,energy_kwh
		FROM samples WHERE ts BETWEEN ? AND ? ORDER BY ts`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DebugRow
	for rows.Next() {
		var r DebugRow
		var ign, cs, plug sql.NullString
		var locked sql.NullBool
		if err := rows.Scan(&r.TS, &ign, &cs, &plug, &r.ChargeKw, &r.SpeedKph, &r.SoC, &r.OdometerKm, &locked, &r.EnergyKwh); err != nil {
			return nil, err
		}
		r.Ignition, r.ChargeState, r.Plug = ign.String, cs.String, plug.String
		if locked.Valid {
			b := locked.Bool
			r.Locked = &b
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ChargingSamples returns the full readings, raw fields included, of the moments a cable was connected and the car said it
// was charging but reported no power: the fast chargers. Few enough to read in full.
func (s *Store) ChargingSamples(from, to int64) ([]*Sample, error) {
	rows, err := s.DB.Query(`SELECT `+sampleCols+` FROM samples WHERE ts BETWEEN ? AND ? AND plug LIKE '%CONNECTED%' AND plug NOT LIKE '%DIS%'
		AND charge_state LIKE '%IN_PROGRESS%' AND (charge_kw IS NULL OR charge_kw < 0.2) ORDER BY ts`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Sample
	for rows.Next() {
		x, err := scanSample(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
