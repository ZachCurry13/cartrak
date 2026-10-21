package store

import (
	"encoding/json"
	"time"
)

const routeCacheDDL = `CREATE TABLE IF NOT EXISTS route_cache (
  trip_id INTEGER PRIMARY KEY, points TEXT NOT NULL, km REAL NOT NULL, created INTEGER NOT NULL)`

// SaveRouteEstimate keeps the estimated road for a trip so the routing service is asked once.
func (s *Store) SaveRouteEstimate(tripID int64, pts [][2]float64, km float64) error {
	if _, err := s.DB.Exec(routeCacheDDL); err != nil {
		return err
	}
	b, _ := json.Marshal(pts)
	_, err := s.DB.Exec(`INSERT OR REPLACE INTO route_cache(trip_id,points,km,created) VALUES(?,?,?,?)`, tripID, string(b), km, time.Now().Unix())
	return err
}

// RouteEstimate is the kept estimate for a trip, or nil.
func (s *Store) RouteEstimate(tripID int64) ([][2]float64, float64) {
	if _, err := s.DB.Exec(routeCacheDDL); err != nil {
		return nil, 0
	}
	var raw string
	var km float64
	if err := s.DB.QueryRow(`SELECT points,km FROM route_cache WHERE trip_id=?`, tripID).Scan(&raw, &km); err != nil {
		return nil, 0
	}
	var pts [][2]float64
	if json.Unmarshal([]byte(raw), &pts) != nil || len(pts) < 2 {
		return nil, 0
	}
	return pts, km
}

// RouteEstimateIDs is which of the given trips already have an estimate.
func (s *Store) RouteEstimateIDs() map[int64]bool {
	out := map[int64]bool{}
	if _, err := s.DB.Exec(routeCacheDDL); err != nil {
		return out
	}
	rows, err := s.DB.Query(`SELECT trip_id FROM route_cache`)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		if rows.Scan(&id) == nil {
			out[id] = true
		}
	}
	return out
}

// DeleteRouteEstimate forgets a trip's estimate (when the trip is deleted or its positions change).
func (s *Store) DeleteRouteEstimate(tripID int64) {
	if _, err := s.DB.Exec(routeCacheDDL); err == nil {
		_, _ = s.DB.Exec(`DELETE FROM route_cache WHERE trip_id=?`, tripID)
	}
}

// SaveNoRoute remembers that a trip's road cannot be estimated (the routing service found a road of a very different length),
// so it is neither offered again nor asked for again. Forgotten with DeleteRouteEstimate.
func (s *Store) SaveNoRoute(tripID int64) error {
	return s.SaveRouteEstimate(tripID, [][2]float64{}, 0)
}

// RouteGaveUp says whether a trip's road was tried and found unusable.
func (s *Store) RouteGaveUp(tripID int64) bool {
	if _, err := s.DB.Exec(routeCacheDDL); err != nil {
		return false
	}
	var raw string
	if s.DB.QueryRow(`SELECT points FROM route_cache WHERE trip_id=?`, tripID).Scan(&raw) != nil {
		return false
	}
	return raw == "[]"
}

// CountNoRoutes is how many trips were tried and found to have no usable road.
func (s *Store) CountNoRoutes() int {
	if _, err := s.DB.Exec(routeCacheDDL); err != nil {
		return 0
	}
	var n int
	_ = s.DB.QueryRow(`SELECT COUNT(*) FROM route_cache WHERE points='[]'`).Scan(&n)
	return n
}

// CountCorrectedTrips is how many trips have a distance the owner corrected.
func (s *Store) CountCorrectedTrips() int {
	var n int
	_ = s.DB.QueryRow(`SELECT COUNT(*) FROM trips WHERE distance_src!=''`).Scan(&n)
	return n
}

// ForgetNoRoutes forgets every trip whose road was given up on, so each is asked about once more.
func (s *Store) ForgetNoRoutes() {
	if _, err := s.DB.Exec(routeCacheDDL); err == nil {
		_, _ = s.DB.Exec(`DELETE FROM route_cache WHERE points='[]'`)
	}
}
