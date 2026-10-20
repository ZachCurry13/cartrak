package store

import (
	"database/sql"
	"strings"
)

type Trip struct {
	ID             int64    `json:"id"`
	StartTS        int64    `json:"startTs"`
	EndTS          *int64   `json:"endTs"`
	StartLat       *float64 `json:"startLat"`
	StartLon       *float64 `json:"startLon"`
	EndLat         *float64 `json:"endLat"`
	EndLon         *float64 `json:"endLon"`
	StartOdo       *float64 `json:"startOdo"`
	EndOdo         *float64 `json:"endOdo"`
	DistanceKm     float64  `json:"distanceKm"`
	StartSoC       *float64 `json:"startSoc"`
	EndSoC         *float64 `json:"endSoc"`
	MaxSpeedKph    float64  `json:"maxSpeedKph"`
	Category       string   `json:"category"`
	CategorySource string   `json:"categorySource"` // default | rule | manual
	RuleID         *int64   `json:"ruleId"`
	Notes          string   `json:"notes"`
	Open           bool     `json:"open"`
	TZ             string   `json:"tz"`       // time zone where the trip started, e.g. America/New_York
	StartKwh       *float64 `json:"startKwh"` // energy left in the battery at the start and end, kWh
	EndKwh         *float64 `json:"endKwh"`
	JourneyID      *int64   `json:"journeyId"` // the journey this trip belongs to, if any
}

const tripCols = `id,start_ts,end_ts,start_lat,start_lon,end_lat,end_lon,start_odo,end_odo,distance_km,
 start_soc,end_soc,max_speed_kph,category,category_source,rule_id,notes,open,tz,start_kwh,end_kwh,journey_id`

func scanTrip(r interface{ Scan(...any) error }) (*Trip, error) {
	var t Trip
	err := r.Scan(&t.ID, &t.StartTS, &t.EndTS, &t.StartLat, &t.StartLon, &t.EndLat, &t.EndLon,
		&t.StartOdo, &t.EndOdo, &t.DistanceKm, &t.StartSoC, &t.EndSoC, &t.MaxSpeedKph,
		&t.Category, &t.CategorySource, &t.RuleID, &t.Notes, &t.Open, &t.TZ, &t.StartKwh, &t.EndKwh, &t.JourneyID)
	return &t, err
}

func (s *Store) OpenTrip() (*Trip, error) {
	t, err := scanTrip(s.DB.QueryRow(`SELECT ` + tripCols + ` FROM trips WHERE open=1 ORDER BY start_ts DESC LIMIT 1`))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return t, err
}

func (s *Store) GetTrip(id int64) (*Trip, error) {
	return scanTrip(s.DB.QueryRow(`SELECT `+tripCols+` FROM trips WHERE id=?`, id))
}

func (s *Store) CreateTrip(t *Trip) error {
	r, err := s.DB.Exec(`INSERT INTO trips(start_ts,start_lat,start_lon,start_odo,start_soc,end_lat,end_lon,
		end_odo,end_soc,category,category_source,rule_id,tz,start_kwh,end_kwh,open) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,1)`,
		t.StartTS, t.StartLat, t.StartLon, t.StartOdo, t.StartSoC, t.StartLat, t.StartLon,
		t.StartOdo, t.StartSoC, t.Category, t.CategorySource, t.RuleID, t.TZ, t.StartKwh, t.StartKwh)
	if err != nil {
		return err
	}
	t.ID, _ = r.LastInsertId()
	t.Open = true
	return nil
}

// UpdateTrip writes the progress fields. Category is only written when the
// current source is not "manual", so a user's choice is never overwritten.
func (s *Store) UpdateTrip(t *Trip) error {
	_, err := s.DB.Exec(`UPDATE trips SET end_ts=?,end_lat=?,end_lon=?,end_odo=?,end_soc=?,end_kwh=?,distance_km=?,
		max_speed_kph=?,open=?,
		category=CASE WHEN category_source='manual' THEN category ELSE ? END,
		rule_id=CASE WHEN category_source='manual' THEN rule_id ELSE ? END,
		category_source=CASE WHEN category_source='manual' THEN 'manual' ELSE ? END
		WHERE id=?`,
		t.EndTS, t.EndLat, t.EndLon, t.EndOdo, t.EndSoC, t.EndKwh, t.DistanceKm, t.MaxSpeedKph, t.Open,
		t.Category, t.RuleID, t.CategorySource, t.ID)
	return err
}

func (s *Store) DeleteTrip(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM trips WHERE id=?`, id)
	s.DeleteRouteEstimate(id) // its estimated road goes with it
	return err
}

// SetTripCategory is a user decision; it always wins over rules.
func (s *Store) SetTripCategory(id int64, cat, notes string) error {
	_, err := s.DB.Exec(`UPDATE trips SET category=?,category_source='manual',rule_id=NULL,notes=? WHERE id=?`, cat, notes, id)
	return err
}

type TripFilter struct {
	From, To int64
	Category string
	Limit    int
}

func (s *Store) ListTrips(f TripFilter) ([]*Trip, error) {
	q := []string{`SELECT ` + tripCols + ` FROM trips WHERE 1=1`}
	var a []any
	if f.From > 0 {
		q = append(q, `AND start_ts>=?`)
		a = append(a, f.From)
	}
	if f.To > 0 {
		q = append(q, `AND start_ts<?`)
		a = append(a, f.To)
	}
	if f.Category != "" {
		q = append(q, `AND category=?`)
		a = append(a, f.Category)
	}
	q = append(q, `ORDER BY start_ts DESC LIMIT ?`)
	if f.Limit <= 0 || f.Limit > 1000 {
		f.Limit = 200
	}
	a = append(a, f.Limit)
	rows, err := s.DB.Query(strings.Join(q, " "), a...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Trip{}
	for rows.Next() {
		t, err := scanTrip(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
