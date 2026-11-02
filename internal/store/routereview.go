package store

import "time"

// route_review keeps why a trip's road could not be worked out, so the list of such trips is still there when the owner comes back to
// Trips (it used to exist only on the page right after the estimates ran, and was gone after opening one trip).
const routeReviewDDL = `CREATE TABLE IF NOT EXISTS route_review (
  trip_id INTEGER PRIMARY KEY, msg TEXT NOT NULL, road_km REAL NOT NULL, odo_km REAL NOT NULL, dismissed INTEGER NOT NULL DEFAULT 0, noted INTEGER NOT NULL)`

// RouteReview is a trip with no road, and why.
type RouteReview struct {
	TripID int64
	Msg    string
	RoadKm float64 // the length of the road found, when one was found but did not fit the odometer
	OdoKm  float64
}

func (s *Store) routeReviewTables() error {
	if _, err := s.DB.Exec(routeCacheDDL); err != nil {
		return err
	}
	_, err := s.DB.Exec(routeReviewDDL)
	return err
}

// NoteRouteReview records that a trip's road was given up on, and why. It is listed again even if it was dismissed before.
func (s *Store) NoteRouteReview(tripID int64, msg string, roadKm, odoKm float64) error {
	if err := s.routeReviewTables(); err != nil {
		return err
	}
	_, err := s.DB.Exec(`INSERT OR REPLACE INTO route_review(trip_id,msg,road_km,odo_km,dismissed,noted) VALUES(?,?,?,?,0,?)`, tripID, msg, roadKm, odoKm, time.Now().Unix())
	return err
}

// DismissRouteReview stops listing these trips: the owner accepts that they stay a straight line.
func (s *Store) DismissRouteReview(ids []int64) error {
	if err := s.routeReviewTables(); err != nil {
		return err
	}
	for _, id := range ids {
		if _, err := s.DB.Exec(`INSERT INTO route_review(trip_id,msg,road_km,odo_km,dismissed,noted) VALUES(?,'',0,0,1,?)
			ON CONFLICT(trip_id) DO UPDATE SET dismissed=1`, id, time.Now().Unix()); err != nil {
			return err
		}
	}
	return nil
}

// SeedRouteReview lists, once, the trips that were given up on before reasons were kept. Their reason is looked up when the trip is opened.
func (s *Store) SeedRouteReview() error {
	if s.GetKV("route_review_seeded") == "1" {
		return nil
	}
	if err := s.routeReviewTables(); err != nil {
		return err
	}
	if _, err := s.DB.Exec(`INSERT OR IGNORE INTO route_review(trip_id,msg,road_km,odo_km,dismissed,noted)
		SELECT trip_id,'',0,0,0,? FROM route_cache WHERE points='[]'`, time.Now().Unix()); err != nil {
		return err
	}
	return s.SetKV("route_review_seeded", "1")
}

// RouteReviews are the trips whose road is still missing and that the owner has not dismissed, newest first.
func (s *Store) RouteReviews() ([]RouteReview, error) {
	if err := s.routeReviewTables(); err != nil {
		return nil, err
	}
	rows, err := s.DB.Query(`SELECT r.trip_id, r.msg, r.road_km, r.odo_km FROM route_review r
		JOIN route_cache c ON c.trip_id=r.trip_id JOIN trips t ON t.id=r.trip_id
		WHERE r.dismissed=0 AND c.points='[]' ORDER BY t.start_ts DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []RouteReview{}
	for rows.Next() {
		var r RouteReview
		if err := rows.Scan(&r.TripID, &r.Msg, &r.RoadKm, &r.OdoKm); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
