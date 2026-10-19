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
}

// DebugRows returns the readings from `from` to `to`, oldest first.
func (s *Store) DebugRows(from, to int64) ([]DebugRow, error) {
	rows, err := s.DB.Query(`SELECT ts,ignition,charge_state,plug,charge_kw,speed_kph,soc_pct,odometer_km,locked
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
		if err := rows.Scan(&r.TS, &ign, &cs, &plug, &r.ChargeKw, &r.SpeedKph, &r.SoC, &r.OdometerKm, &locked); err != nil {
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
