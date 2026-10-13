package store

import "fmt"

// Day is one day of a trended value: its average, lowest and highest reading.
type Day struct {
	TS  int64   `json:"ts"` // start of the day, unix seconds (UTC)
	Avg float64 `json:"avg"`
	Min float64 `json:"min"`
	Max float64 `json:"max"`
	N   int     `json:"n"`
}

// trended lists the values the Vehicle health page can chart, and the column holding each.
var trended = map[string]string{
	"battery12v": "battery12v_v", "hvVolts": "hv_volts", "capacityKwh": "capacity_kwh",
	"oilLifePct": "oil_pct", "fuelPct": "fuel_pct", "socPct": "soc_pct",
}

// DailySeries returns one row per day, oldest first, for a trended value since from (unix seconds).
// Only names in trended are accepted, so the column name never comes from the caller.
func (s *Store) DailySeries(name string, from int64) ([]Day, error) {
	col, ok := trended[name]
	if !ok {
		return nil, fmt.Errorf("unknown series %q", name)
	}
	rows, err := s.DB.Query(`SELECT (ts/86400)*86400 AS d, AVG(`+col+`), MIN(`+col+`), MAX(`+col+`), COUNT(*)
		FROM samples WHERE ts >= ? AND `+col+` IS NOT NULL GROUP BY d ORDER BY d`, from)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Day{}
	for rows.Next() {
		var d Day
		if err := rows.Scan(&d.TS, &d.Avg, &d.Min, &d.Max, &d.N); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
