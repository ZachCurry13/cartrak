package store

import (
	"database/sql"
	"time"
)

// BugReport is a note about a problem with a text copy of the screen it was about: the page's words, the device, layout facts and
// the debug report. Reports stay in the app until cleared.
type BugReport struct {
	ID      int64  `json:"id"`
	TS      int64  `json:"ts"`
	Version string `json:"version"`
	Note    string `json:"note"`
	Page    string `json:"page"`
	Device  string `json:"device"`
	Text    string `json:"text"`
	Layout  string `json:"layout"`
	Debug   string `json:"debug"`
}

const bugDDL = `CREATE TABLE IF NOT EXISTS bug_reports (id INTEGER PRIMARY KEY, ts INTEGER NOT NULL, version TEXT NOT NULL DEFAULT '', note TEXT NOT NULL DEFAULT '',
 page TEXT NOT NULL DEFAULT '', device TEXT NOT NULL DEFAULT '', text TEXT NOT NULL DEFAULT '', layout TEXT NOT NULL DEFAULT '', debug TEXT NOT NULL DEFAULT '');
CREATE TABLE IF NOT EXISTS report_links (id INTEGER PRIMARY KEY, token_hash TEXT NOT NULL, created INTEGER NOT NULL, expires INTEGER NOT NULL, last_used INTEGER NOT NULL DEFAULT 0, opens INTEGER NOT NULL DEFAULT 0)`

func (s *Store) bugInit() error { _, err := s.DB.Exec(bugDDL); return err }

// AddBugReport keeps a report and returns it with its number.
func (s *Store) AddBugReport(r BugReport) (BugReport, error) {
	if err := s.bugInit(); err != nil {
		return r, err
	}
	res, err := s.DB.Exec(`INSERT INTO bug_reports(ts,version,note,page,device,text,layout,debug) VALUES(?,?,?,?,?,?,?,?)`, r.TS, r.Version, r.Note, r.Page, r.Device, r.Text, r.Layout, r.Debug)
	if err != nil {
		return r, err
	}
	r.ID, _ = res.LastInsertId()
	return r, nil
}

// BugReports lists the reports newest first. With full they carry their text; otherwise only what a list needs.
func (s *Store) BugReports(full bool) ([]BugReport, error) {
	if err := s.bugInit(); err != nil {
		return nil, err
	}
	q := `SELECT id,ts,version,note,page,device,'','','' FROM bug_reports ORDER BY ts DESC, id DESC`
	if full {
		q = `SELECT id,ts,version,note,page,device,text,layout,debug FROM bug_reports ORDER BY ts DESC, id DESC`
	}
	rows, err := s.DB.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []BugReport{}
	for rows.Next() {
		var r BugReport
		if err := rows.Scan(&r.ID, &r.TS, &r.Version, &r.Note, &r.Page, &r.Device, &r.Text, &r.Layout, &r.Debug); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// BugReport is one report with everything in it, or nil.
func (s *Store) BugReport(id int64) (*BugReport, error) {
	if err := s.bugInit(); err != nil {
		return nil, err
	}
	var r BugReport
	err := s.DB.QueryRow(`SELECT id,ts,version,note,page,device,text,layout,debug FROM bug_reports WHERE id=?`, id).Scan(&r.ID, &r.TS, &r.Version, &r.Note, &r.Page, &r.Device, &r.Text, &r.Layout, &r.Debug)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &r, err
}

// DeleteBugReports removes one report (id > 0), or every report older than a time (id = 0; before 0 removes all).
func (s *Store) DeleteBugReports(id, before int64) (int64, error) {
	if err := s.bugInit(); err != nil {
		return 0, err
	}
	var res sql.Result
	var err error
	if id > 0 {
		res, err = s.DB.Exec(`DELETE FROM bug_reports WHERE id=?`, id)
	} else if before > 0 {
		res, err = s.DB.Exec(`DELETE FROM bug_reports WHERE ts<?`, before)
	} else {
		res, err = s.DB.Exec(`DELETE FROM bug_reports`)
	}
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// ReportLink is the one link that opens the reports as text, made on demand, expiring and revocable.
type ReportLink struct {
	Created  int64 `json:"created"`
	Expires  int64 `json:"expires"`
	LastUsed int64 `json:"lastUsed"`
	Opens    int   `json:"opens"`
}

// NewReportLink replaces any earlier link with one for the given token hash.
func (s *Store) NewReportLink(hash string, expires int64) error {
	if err := s.bugInit(); err != nil {
		return err
	}
	if _, err := s.DB.Exec(`DELETE FROM report_links`); err != nil {
		return err
	}
	_, err := s.DB.Exec(`INSERT INTO report_links(token_hash,created,expires) VALUES(?,?,?)`, hash, time.Now().Unix(), expires)
	return err
}

// RevokeReportLink forgets the link.
func (s *Store) RevokeReportLink() error {
	if err := s.bugInit(); err != nil {
		return err
	}
	_, err := s.DB.Exec(`DELETE FROM report_links`)
	return err
}

// ReportLinkOK says whether the hash is the current link and it has not expired, and counts the opening.
func (s *Store) ReportLinkOK(hash string, now int64) bool {
	if s.bugInit() != nil {
		return false
	}
	res, err := s.DB.Exec(`UPDATE report_links SET last_used=?, opens=opens+1 WHERE token_hash=? AND (expires=0 OR expires>?)`, now, hash, now)
	if err != nil {
		return false
	}
	n, _ := res.RowsAffected()
	return n > 0
}

// CurrentReportLink describes the link now in use, or nil.
func (s *Store) CurrentReportLink() *ReportLink {
	if s.bugInit() != nil {
		return nil
	}
	var l ReportLink
	if err := s.DB.QueryRow(`SELECT created,expires,last_used,opens FROM report_links LIMIT 1`).Scan(&l.Created, &l.Expires, &l.LastUsed, &l.Opens); err != nil {
		return nil
	}
	return &l
}
