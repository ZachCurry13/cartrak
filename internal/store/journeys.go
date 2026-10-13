package store

import (
	"database/sql"
	"strings"
	"time"
)

// Journey is a group of trips shown as one: a road trip, or a work day with several stops.
type Journey struct {
	ID         int64   `json:"id"`
	Name       string  `json:"name"`
	Created    int64   `json:"created"`
	Trips      int     `json:"trips"`
	StartTS    int64   `json:"startTs"`
	EndTS      int64   `json:"endTs"`
	DistanceKm float64 `json:"distanceKm"`
}

func (s *Store) CreateJourney(name string) (int64, error) {
	r, err := s.DB.Exec(`INSERT INTO journeys(name,created) VALUES(?,?)`, name, time.Now().Unix())
	if err != nil {
		return 0, err
	}
	return r.LastInsertId()
}

const journeyQuery = `SELECT j.id,j.name,j.created,COUNT(t.id),COALESCE(MIN(t.start_ts),0),COALESCE(MAX(t.end_ts),0),COALESCE(SUM(t.distance_km),0)
	FROM journeys j LEFT JOIN trips t ON t.journey_id=j.id`

func scanJourney(r interface{ Scan(...any) error }) (*Journey, error) {
	var j Journey
	err := r.Scan(&j.ID, &j.Name, &j.Created, &j.Trips, &j.StartTS, &j.EndTS, &j.DistanceKm)
	return &j, err
}

// ListJourneys returns every journey with its totals, newest first.
func (s *Store) ListJourneys() ([]*Journey, error) {
	rows, err := s.DB.Query(journeyQuery + ` GROUP BY j.id ORDER BY COALESCE(MIN(t.start_ts),j.created) DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Journey{}
	for rows.Next() {
		j, err := scanJourney(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

func (s *Store) GetJourney(id int64) (*Journey, error) {
	j, err := scanJourney(s.DB.QueryRow(journeyQuery+` WHERE j.id=? GROUP BY j.id`, id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return j, err
}

// JourneyTrips returns the journey's trips, oldest first.
func (s *Store) JourneyTrips(id int64) ([]*Trip, error) {
	rows, err := s.DB.Query(`SELECT `+tripCols+` FROM trips WHERE journey_id=? ORDER BY start_ts`, id)
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

// TripsByIDs returns the trips with these ids, oldest first.
func (s *Store) TripsByIDs(ids []int64) ([]*Trip, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	rows, err := s.DB.Query(`SELECT `+tripCols+` FROM trips WHERE id IN (`+strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")+`) ORDER BY start_ts`, args...)
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

// SetTripsJourney puts trips into a journey (id > 0) or takes them out of any journey (id == 0).
func (s *Store) SetTripsJourney(ids []int64, journey int64) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, id := range ids {
		var j any
		if journey > 0 {
			j = journey
		}
		if _, err := tx.Exec(`UPDATE trips SET journey_id=? WHERE id=?`, j, id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) RenameJourney(id int64, name string) error {
	_, err := s.DB.Exec(`UPDATE journeys SET name=? WHERE id=?`, name, id)
	return err
}

// DeleteJourney ungroups its trips (they stay) and removes the journey.
func (s *Store) DeleteJourney(id int64) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE trips SET journey_id=NULL WHERE journey_id=?`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM journeys WHERE id=?`, id); err != nil {
		return err
	}
	return tx.Commit()
}
