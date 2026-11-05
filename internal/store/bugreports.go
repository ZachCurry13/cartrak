package store

import (
	"database/sql"
	"time"
)

// BugReport is a note about a problem with a text copy of the screen it was about: the page's words, the device, layout facts and
// the debug report. Reports stay in the app until cleared.
type BugReport struct {
	ID       int64  `json:"id"`
	TS       int64  `json:"ts"`
	Version  string `json:"version"`
	Note     string `json:"note"`
	Page     string `json:"page"`
	Device   string `json:"device"`
	Text     string `json:"text"`
	Layout   string `json:"layout"`
	Debug    string `json:"debug"`
	HTML     string `json:"html,omitempty"` // the page as it looked: its markup and styles, without scripts
	HasPage  bool   `json:"hasPage"`        // there is a page copy to open
	HasDebug bool   `json:"hasDebug"`       // there is a debug report to open
	Fixed    int64  `json:"fixed"`          // when it was marked fixed, or 0 while it is open
}

const bugDDL = `CREATE TABLE IF NOT EXISTS bug_reports (id INTEGER PRIMARY KEY, ts INTEGER NOT NULL, version TEXT NOT NULL DEFAULT '', note TEXT NOT NULL DEFAULT '',
 page TEXT NOT NULL DEFAULT '', device TEXT NOT NULL DEFAULT '', text TEXT NOT NULL DEFAULT '', layout TEXT NOT NULL DEFAULT '', debug TEXT NOT NULL DEFAULT '');
CREATE TABLE IF NOT EXISTS report_links (id INTEGER PRIMARY KEY, token_hash TEXT NOT NULL, created INTEGER NOT NULL, expires INTEGER NOT NULL, last_used INTEGER NOT NULL DEFAULT 0, opens INTEGER NOT NULL DEFAULT 0)`

func (s *Store) bugInit() error {
	if _, err := s.DB.Exec(bugDDL); err != nil {
		return err
	}
	_, _ = s.DB.Exec(`ALTER TABLE bug_reports ADD COLUMN html TEXT NOT NULL DEFAULT ''`) // already there on later runs
	_, _ = s.DB.Exec(`ALTER TABLE report_links ADD COLUMN token TEXT NOT NULL DEFAULT ''`)
	_, _ = s.DB.Exec(`ALTER TABLE bug_reports ADD COLUMN fixed INTEGER NOT NULL DEFAULT 0`)
	return nil
}

// AddBugReport keeps a report and returns it with its number.
func (s *Store) AddBugReport(r BugReport) (BugReport, error) {
	if err := s.bugInit(); err != nil {
		return r, err
	}
	res, err := s.DB.Exec(`INSERT INTO bug_reports(ts,version,note,page,device,text,layout,debug,html) VALUES(?,?,?,?,?,?,?,?,?)`, r.TS, r.Version, r.Note, r.Page, r.Device, r.Text, r.Layout, r.Debug, r.HTML)
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
	q := `SELECT id,ts,version,note,page,device,'','','',html<>'',fixed FROM bug_reports ORDER BY ts DESC, id DESC`
	if full {
		q = `SELECT id,ts,version,note,page,device,text,layout,debug,html<>'',fixed FROM bug_reports ORDER BY ts DESC, id DESC`
	}
	rows, err := s.DB.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []BugReport{}
	for rows.Next() {
		var r BugReport
		if err := rows.Scan(&r.ID, &r.TS, &r.Version, &r.Note, &r.Page, &r.Device, &r.Text, &r.Layout, &r.Debug, &r.HasPage, &r.Fixed); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// BugReportBriefs lists the reports newest first with what a short index needs (the note, page, device and layout facts) and
// whether each has a page copy and a debug report, but not their bodies.
func (s *Store) BugReportBriefs() ([]BugReport, error) {
	if err := s.bugInit(); err != nil {
		return nil, err
	}
	rows, err := s.DB.Query(`SELECT id,ts,version,note,page,device,layout,html<>'',debug<>'',fixed FROM bug_reports ORDER BY ts DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []BugReport{}
	for rows.Next() {
		var r BugReport
		if err := rows.Scan(&r.ID, &r.TS, &r.Version, &r.Note, &r.Page, &r.Device, &r.Layout, &r.HasPage, &r.HasDebug, &r.Fixed); err != nil {
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
	err := s.DB.QueryRow(`SELECT id,ts,version,note,page,device,text,layout,debug,html,fixed FROM bug_reports WHERE id=?`, id).Scan(&r.ID, &r.TS, &r.Version, &r.Note, &r.Page, &r.Device, &r.Text, &r.Layout, &r.Debug, &r.HTML, &r.Fixed)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	r.HasPage, r.HasDebug = r.HTML != "", r.Debug != ""
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
	Token    string `json:"token,omitempty"` // kept so the link can be shown again; only the owner reads it
	Created  int64  `json:"created"`
	Expires  int64  `json:"expires"`
	LastUsed int64  `json:"lastUsed"`
	Opens    int    `json:"opens"`
}

// NewReportLink replaces any earlier link with one for the given token hash.
func (s *Store) NewReportLink(token, hash string, expires int64) error {
	if err := s.bugInit(); err != nil {
		return err
	}
	if _, err := s.DB.Exec(`DELETE FROM report_links`); err != nil {
		return err
	}
	_, err := s.DB.Exec(`INSERT INTO report_links(token_hash,token,created,expires) VALUES(?,?,?,?)`, hash, token, time.Now().Unix(), expires)
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
	if err := s.DB.QueryRow(`SELECT token,created,expires,last_used,opens FROM report_links LIMIT 1`).Scan(&l.Token, &l.Created, &l.Expires, &l.LastUsed, &l.Opens); err != nil {
		return nil
	}
	return &l
}

// SetBugReportFixed marks a report fixed (at the time given) or open again (0).
func (s *Store) SetBugReportFixed(id, at int64) error {
	if err := s.bugInit(); err != nil {
		return err
	}
	_, err := s.DB.Exec(`UPDATE bug_reports SET fixed=? WHERE id=?`, at, id)
	return err
}

// DeleteFixedBugReports removes the reports marked fixed before a time (0 removes every fixed one).
func (s *Store) DeleteFixedBugReports(before int64) (int64, error) {
	if err := s.bugInit(); err != nil {
		return 0, err
	}
	var res sql.Result
	var err error
	if before > 0 {
		res, err = s.DB.Exec(`DELETE FROM bug_reports WHERE fixed>0 AND fixed<?`, before)
	} else {
		res, err = s.DB.Exec(`DELETE FROM bug_reports WHERE fixed>0`)
	}
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// BugReportCount is how many reports are kept, counted without reading any of them.
func (s *Store) BugReportCount() int {
	if s.bugInit() != nil {
		return 0
	}
	var n int
	_ = s.DB.QueryRow(`SELECT count(*) FROM bug_reports`).Scan(&n)
	return n
}

// HasReportLink says whether the hash is this car's current report link, without counting an opening, so the server can find
// which car a link belongs to.
func (s *Store) HasReportLink(hash string) bool {
	if s.bugInit() != nil {
		return false
	}
	var n int
	_ = s.DB.QueryRow(`SELECT count(*) FROM report_links WHERE token_hash=?`, hash).Scan(&n)
	return n > 0
}
