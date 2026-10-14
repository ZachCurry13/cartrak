package store

import "time"

// AlertLogEntry is one alert CarTrak decided to send, and who it reached.
type AlertLogEntry struct {
	ID      int64  `json:"id"`
	TS      int64  `json:"ts"`
	Kind    string `json:"kind"`
	Title   string `json:"title"`
	Body    string `json:"body"`
	Wanted  int    `json:"wanted"`  // devices that asked for this kind and were not in quiet hours
	Sent    int    `json:"sent"`    // of those, how many the push service accepted
	Quiet   int    `json:"quiet"`   // devices skipped because of quiet hours
	Devices int    `json:"devices"` // devices registered at the time
}

// LogAlert records an alert and keeps the newest 200.
func (s *Store) LogAlert(e AlertLogEntry) error {
	if e.TS == 0 {
		e.TS = time.Now().Unix()
	}
	if _, err := s.DB.Exec(`INSERT INTO alert_log(ts,kind,title,body,wanted,sent,quiet,devices) VALUES(?,?,?,?,?,?,?,?)`, e.TS, e.Kind, e.Title, e.Body, e.Wanted, e.Sent, e.Quiet, e.Devices); err != nil {
		return err
	}
	_, err := s.DB.Exec(`DELETE FROM alert_log WHERE id <= (SELECT MAX(id) FROM alert_log) - 200`)
	return err
}

func (s *Store) RecentAlerts(limit int) ([]AlertLogEntry, error) {
	rows, err := s.DB.Query(`SELECT id,ts,kind,title,body,wanted,sent,quiet,devices FROM alert_log ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AlertLogEntry{}
	for rows.Next() {
		var e AlertLogEntry
		if err := rows.Scan(&e.ID, &e.TS, &e.Kind, &e.Title, &e.Body, &e.Wanted, &e.Sent, &e.Quiet, &e.Devices); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
