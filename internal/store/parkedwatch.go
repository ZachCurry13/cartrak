package store

import (
	"strconv"
	"time"
)

// ParkedEvent is something the parked watch noticed: a motion reading above the bump level, or an alarm state.
type ParkedEvent struct {
	ID    int64   `json:"id"`
	TS    int64   `json:"ts"` // when the car reported it
	Kind  string  `json:"kind"`
	Value float64 `json:"value"`
	Note  string  `json:"note"`
}

const (
	kvWatchChecked = "pw_checked"
	kvWatchMax     = "pw_max"
	kvWatchMaxTS   = "pw_max_ts"
	kvWatchSince   = "pw_since"
)

// AddParkedEvent records an event and keeps the newest 300.
func (s *Store) AddParkedEvent(e ParkedEvent) error {
	if _, err := s.DB.Exec(`INSERT INTO parked_events(ts,kind,value,note) VALUES(?,?,?,?)`, e.TS, e.Kind, e.Value, e.Note); err != nil {
		return err
	}
	_, err := s.DB.Exec(`DELETE FROM parked_events WHERE id <= (SELECT MAX(id) FROM parked_events) - 300`)
	return err
}

func (s *Store) RecentParkedEvents(limit int) ([]ParkedEvent, error) {
	rows, err := s.DB.Query(`SELECT id,ts,kind,value,note FROM parked_events ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ParkedEvent{}
	for rows.Next() {
		var e ParkedEvent
		if err := rows.Scan(&e.ID, &e.TS, &e.Kind, &e.Value, &e.Note); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// NoteParkedReading counts one parked reading that was checked and remembers the strongest motion seen.
func (s *Store) NoteParkedReading(motion float64, ts int64) {
	n, _ := strconv.ParseInt(s.GetKV(kvWatchChecked), 10, 64)
	_ = s.SetKV(kvWatchChecked, strconv.FormatInt(n+1, 10))
	if s.GetKV(kvWatchSince) == "" {
		_ = s.SetKV(kvWatchSince, strconv.FormatInt(time.Now().Unix(), 10))
	}
	if cur, _ := strconv.ParseFloat(s.GetKV(kvWatchMax), 64); motion > cur {
		_ = s.SetKV(kvWatchMax, strconv.FormatFloat(motion, 'f', 3, 64))
		_ = s.SetKV(kvWatchMaxTS, strconv.FormatInt(ts, 10))
	}
}

// ParkedWatchStats are the running totals shown on the Parked watch page.
type ParkedWatchStats struct {
	Checked int64   `json:"checked"`
	MaxSeen float64 `json:"maxSeen"`
	MaxTS   int64   `json:"maxTs"`
	Since   int64   `json:"since"`
}

func (s *Store) ParkedWatchStats() ParkedWatchStats {
	var st ParkedWatchStats
	st.Checked, _ = strconv.ParseInt(s.GetKV(kvWatchChecked), 10, 64)
	st.MaxSeen, _ = strconv.ParseFloat(s.GetKV(kvWatchMax), 64)
	st.MaxTS, _ = strconv.ParseInt(s.GetKV(kvWatchMaxTS), 10, 64)
	st.Since, _ = strconv.ParseInt(s.GetKV(kvWatchSince), 10, 64)
	return st
}

// ResetParkedWatch clears the totals and the list, to start a fresh trial.
func (s *Store) ResetParkedWatch() error {
	for _, k := range []string{kvWatchChecked, kvWatchMax, kvWatchMaxTS, kvWatchSince} {
		_ = s.SetKV(k, "")
	}
	_, err := s.DB.Exec(`DELETE FROM parked_events`)
	return err
}
