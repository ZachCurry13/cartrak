package store

import (
	"fmt"
	"sort"
)

// Day is one day of a trended value: its average, lowest and highest reading.
type Day struct {
	TS  int64   `json:"ts"` // start of the day, unix seconds (UTC)
	Avg float64 `json:"avg"`
	Min float64 `json:"min"`
	Max float64 `json:"max"`
	N   int     `json:"n"`
}

// trended lists the values the Vehicle health page can chart, and the column of the trends
// table holding each.
var trended = map[string]string{
	"battery12v": "battery12v", "hvVolts": "hv_volts", "capacityKwh": "capacity_kwh",
	"oilLifePct": "oil_pct", "fuelPct": "fuel_pct", "socPct": "soc",
}

// DailySeries returns one row per day, oldest first, for a trended value since from (unix seconds).
// Only names in trended are accepted, so the column name never comes from the caller. It reads the
// small trends table, never the large sample rows.
func (s *Store) DailySeries(name string, from int64) ([]Day, error) {
	col, ok := trended[name]
	if !ok {
		return nil, fmt.Errorf("unknown series %q", name)
	}
	rows, err := s.DB.Query(`SELECT (ts/86400)*86400 AS d, AVG(`+col+`), MIN(`+col+`), MAX(`+col+`), COUNT(*)
		FROM trends WHERE ts >= ? AND `+col+` IS NOT NULL GROUP BY d ORDER BY d`, from)
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

// DailyMedian is DailySeries with the middle reading of each day in Avg instead of the average. A value the car estimates, such as
// the battery capacity, wanders with the temperature and the charge level, and a few odd readings pull an average around; the
// middle reading is steadier.
func (s *Store) DailyMedian(name string, from int64) ([]Day, error) {
	col, ok := trended[name]
	if !ok {
		return nil, fmt.Errorf("unknown series %q", name)
	}
	rows, err := s.DB.Query(`SELECT (ts/86400)*86400, `+col+` FROM trends WHERE ts >= ? AND `+col+` IS NOT NULL ORDER BY ts`, from)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Day{}
	var vals []float64
	flush := func() {
		if len(vals) == 0 {
			return
		}
		d := &out[len(out)-1]
		sort.Float64s(vals)
		d.N, d.Min, d.Max = len(vals), vals[0], vals[len(vals)-1]
		d.Avg = vals[len(vals)/2]
		if len(vals)%2 == 0 {
			d.Avg = (vals[len(vals)/2-1] + vals[len(vals)/2]) / 2
		}
		vals = vals[:0]
	}
	for rows.Next() {
		var day int64
		var v float64
		if err := rows.Scan(&day, &v); err != nil {
			return nil, err
		}
		if len(out) == 0 || out[len(out)-1].TS != day {
			flush()
			out = append(out, Day{TS: day})
		}
		vals = append(vals, v)
	}
	flush()
	return out, rows.Err()
}
