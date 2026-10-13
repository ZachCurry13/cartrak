package store

import "time"

// GetPlace returns a remembered city for a rounded place (see geocode.Key).
func (s *Store) GetPlace(key string) (string, bool) {
	var city string
	if err := s.DB.QueryRow(`SELECT city FROM places WHERE k=?`, key).Scan(&city); err != nil {
		return "", false
	}
	return city, true
}

// SetPlace remembers the city for a place. An empty city is remembered too ("nothing known here").
func (s *Store) SetPlace(key, city string) error {
	_, err := s.DB.Exec(`INSERT INTO places(k,city,ts) VALUES(?,?,?) ON CONFLICT(k) DO UPDATE SET city=excluded.city, ts=excluded.ts`, key, city, time.Now().Unix())
	return err
}
