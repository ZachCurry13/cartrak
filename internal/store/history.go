package store

import "database/sql"

// SampleAt returns the newest reading at or before ts (the car as it was at that moment), or nil
// when nothing was recorded by then. It reads one row.
func (s *Store) SampleAt(ts int64) (*Sample, error) {
	x, err := scanSample(s.DB.QueryRow(`SELECT `+sampleCols+` FROM samples WHERE ts<=? ORDER BY ts DESC LIMIT 1`, ts))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return x, err
}

// NeighborTS gives the time of the reading just before and just after ts (0 when there is none),
// for stepping through history. It reads only the index.
func (s *Store) NeighborTS(ts int64) (prev, next int64) {
	_ = s.DB.QueryRow(`SELECT ts FROM samples WHERE ts<? ORDER BY ts DESC LIMIT 1`, ts).Scan(&prev)
	_ = s.DB.QueryRow(`SELECT ts FROM samples WHERE ts>? ORDER BY ts ASC LIMIT 1`, ts).Scan(&next)
	return prev, next
}

// FirstSampleTS is when the oldest reading was recorded (0 if none).
func (s *Store) FirstSampleTS() int64 {
	var ts int64
	_ = s.DB.QueryRow(`SELECT COALESCE(MIN(ts),0) FROM samples`).Scan(&ts)
	return ts
}

// TimePoint is one moment on a timeline: a few numbers and what the car was doing.
type TimePoint struct {
	TS       int64    `json:"ts"`
	SoC      *float64 `json:"soc"`
	RangeKm  *float64 `json:"rangeKm"`
	Odo      *float64 `json:"odometerKm"`
	SpeedKph *float64 `json:"speedKph"`
	OutsideC *float64 `json:"outsideC"`
	Driving  bool     `json:"driving"`
	Charging bool     `json:"charging"`
}

// Timeline returns the readings between from and to (unix seconds) from the small trends table,
// thinned to at most max points (always keeping the first and last). It never reads the big rows.
func (s *Store) Timeline(from, to int64, max int) ([]TimePoint, error) {
	rows, err := s.DB.Query(`SELECT ts,soc,range_km,odometer_km,speed_kph,outside_c,driving,charging FROM trends WHERE ts BETWEEN ? AND ? ORDER BY ts`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var all []TimePoint
	for rows.Next() {
		var p TimePoint
		var d, c int
		if err := rows.Scan(&p.TS, &p.SoC, &p.RangeKm, &p.Odo, &p.SpeedKph, &p.OutsideC, &d, &c); err != nil {
			return nil, err
		}
		p.Driving, p.Charging = d == 1, c == 1
		all = append(all, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if max < 2 || len(all) <= max {
		return append([]TimePoint{}, all...), nil
	}
	out := make([]TimePoint, 0, max)
	step := float64(len(all)-1) / float64(max-1)
	for i := 0; i < max; i++ {
		out = append(out, all[int(float64(i)*step+0.5)])
	}
	return out, nil
}
