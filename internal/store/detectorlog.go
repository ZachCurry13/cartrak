package store

import "time"

const detectorLogDDL = `CREATE TABLE IF NOT EXISTS detector_log (id INTEGER PRIMARY KEY, ts INTEGER NOT NULL, kind TEXT NOT NULL, note TEXT NOT NULL)`

// LogEntry is one decision CarTrak made that leaves no trace in the records themselves: a blip thrown away, a paused
// charge picked up again, an alert held back. The Debug report lists them so odd behavior can be explained.
type LogEntry struct {
	TS   int64  `json:"ts"`
	Kind string `json:"kind"`
	Note string `json:"note"`
}

// LogDecision keeps a short note. Only the latest few hundred are kept.
func (s *Store) LogDecision(kind, note string) {
	if _, err := s.DB.Exec(detectorLogDDL); err != nil {
		return
	}
	if len(note) > 300 {
		note = note[:300]
	}
	_, _ = s.DB.Exec(`INSERT INTO detector_log(ts,kind,note) VALUES(?,?,?)`, time.Now().Unix(), kind, note)
	_, _ = s.DB.Exec(`DELETE FROM detector_log WHERE id <= (SELECT MAX(id) FROM detector_log) - 400`)
}

// RecentDecisions returns the notes from `from` on, oldest first.
func (s *Store) RecentDecisions(from int64) []LogEntry {
	out := []LogEntry{}
	if _, err := s.DB.Exec(detectorLogDDL); err != nil {
		return out
	}
	rows, err := s.DB.Query(`SELECT ts,kind,note FROM detector_log WHERE ts>=? ORDER BY id`, from)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var e LogEntry
		if rows.Scan(&e.TS, &e.Kind, &e.Note) == nil {
			out = append(out, e)
		}
	}
	return out
}
