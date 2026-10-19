package store

import "database/sql"

const weatherSchema = `CREATE TABLE IF NOT EXISTS weather_cache (
  cell TEXT NOT NULL, day TEXT NOT NULL, body TEXT NOT NULL, fetched INTEGER NOT NULL, PRIMARY KEY(cell, day))`

// WeatherGet returns the saved hourly wind for a map cell and a UTC day (YYYY-MM-DD), or "" when there is none.
func (s *Store) WeatherGet(cell, day string) string {
	if _, err := s.DB.Exec(weatherSchema); err != nil {
		return ""
	}
	var body sql.NullString
	_ = s.DB.QueryRow(`SELECT body FROM weather_cache WHERE cell=? AND day=?`, cell, day).Scan(&body)
	return body.String
}

// WeatherPut saves the hourly wind for a cell and day.
func (s *Store) WeatherPut(cell, day, body string, now int64) error {
	if _, err := s.DB.Exec(weatherSchema); err != nil {
		return err
	}
	_, err := s.DB.Exec(`INSERT OR REPLACE INTO weather_cache(cell,day,body,fetched) VALUES(?,?,?,?)`, cell, day, body, now)
	return err
}
