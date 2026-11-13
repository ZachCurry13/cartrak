package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

const (
	reportKeepKey  = "report_keep_days"
	reportMaxKept  = 200
	reportBodyMax  = 5 << 20
	reportHTMLMax  = 2 << 20
	reportDebugMax = 1 << 20
	reportPageMax  = 1 << 20 // what one link answers with
)

// pruneReports forgets reports older than the keep setting (0 keeps them all).
func (s *Server) pruneReports() {
	// a fixed report goes after the delay the owner chose (at once, a day or a week)
	before := time.Now().AddDate(0, 0, -s.fixedKeepDays()).Unix()
	if s.fixedKeepDays() == 0 {
		before = time.Now().Unix() + 1
	}
	_, _ = s.S.DeleteFixedBugReports(before)
	if d, _ := strconv.Atoi(s.S.GetKV(reportKeepKey)); d > 0 {
		_, _ = s.S.DeleteBugReports(0, time.Now().AddDate(0, 0, -d).Unix())
	}
}

// listReports: GET /api/bug-reports. What the Bug reports page shows: the reports, how long they are kept, and the link if one is open.
func (s *Server) listReports(w http.ResponseWriter, r *http.Request) {
	s.pruneReports()
	l, err := s.S.BugReports(false)
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	keep, _ := strconv.Atoi(s.S.GetKV(reportKeepKey))
	link := s.S.CurrentReportLink()
	url := ""
	if link != nil && link.Token != "" {
		url = s.publicBase() + "/r/" + link.Token
		link.Token = ""
	}
	ok(w, map[string]any{"reports": l, "keepDays": keep, "fixedDays": s.fixedKeepDays(), "link": link, "linkUrl": url, "addressKnown": s.publicBase() != ""})
}

// addReport: POST /api/bug-reports. A report is text: the note, what the page said, the device and screen, facts about the layout and
// the debug report. Admin only.
func (s *Server) addReport(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, reportBodyMax)
	var in store.BugReport
	if decode(r, &in) != nil {
		fail(w, 413, "that report is too large")
		return
	}
	if len(in.Note) > 4000 || len(in.Page) > 200 || len(in.Device) > 1000 || len(in.Text) > 60000 || len(in.Layout) > 30000 || len(in.Debug) > reportDebugMax || len(in.HTML) > reportHTMLMax {
		fail(w, 400, "part of that report is too long")
		return
	}
	if s.S.BugReportCount() >= reportMaxKept {
		fail(w, 400, "There are a lot of reports: clear some first")
		return
	}
	in.ID, in.TS, in.Version = 0, time.Now().Unix(), s.Version
	out, err := s.S.AddBugReport(in)
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	s.pruneReports()
	ok(w, map[string]any{"id": out.ID})
}

// oneReport: GET /api/bug-reports/{id}.
func (s *Server) oneReport(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	rep, err := s.S.BugReport(id)
	if err != nil || rep == nil {
		fail(w, 404, "no such report")
		return
	}
	rep.HTML = ""
	ok(w, rep)
}

// deleteReport: DELETE /api/bug-reports/{id}.
func (s *Server) deleteReport(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if id <= 0 {
		fail(w, 400, "no such report")
		return
	}
	if _, err := s.S.DeleteBugReports(id, 0); err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, map[string]bool{"ok": true})
}

// clearReports: POST /api/bug-reports/clear {"olderThanDays": 30}; 0 clears every report.
func (s *Server) clearReports(w http.ResponseWriter, r *http.Request) {
	var in struct {
		OlderThanDays int `json:"olderThanDays"`
	}
	if decode(r, &in) != nil || in.OlderThanDays < 0 || in.OlderThanDays > 3650 {
		fail(w, 400, "give a number of days, or 0 for all")
		return
	}
	before := int64(0)
	if in.OlderThanDays > 0 {
		before = time.Now().AddDate(0, 0, -in.OlderThanDays).Unix()
	}
	n, err := s.S.DeleteBugReports(0, before)
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, map[string]int64{"removed": n})
}

// keepReports: PUT /api/bug-reports/keep {"days": 30}; 0 keeps them until cleared.
func (s *Server) keepReports(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Days      *int `json:"days"`      // how long reports are kept: 0 (until cleared), 30, 60 or 90
		FixedDays *int `json:"fixedDays"` // how long a report marked fixed is kept: 0 (clear it at once), 1 or 7
	}
	if decode(r, &in) != nil || (in.Days == nil && in.FixedDays == nil) ||
		(in.Days != nil && *in.Days != 0 && *in.Days != 30 && *in.Days != 60 && *in.Days != 90) ||
		(in.FixedDays != nil && *in.FixedDays != 0 && *in.FixedDays != 1 && *in.FixedDays != 7) {
		fail(w, 400, "keep reports for 0 (until cleared), 30, 60 or 90 days; clear a fixed one at once (0), after 1 day or after 7")
		return
	}
	if in.Days != nil {
		_ = s.S.SetKV(reportKeepKey, strconv.Itoa(*in.Days))
	}
	if in.FixedDays != nil {
		_ = s.S.SetKV(fixedKeepKey, strconv.Itoa(*in.FixedDays))
	}
	s.pruneReports()
	keep, _ := strconv.Atoi(s.S.GetKV(reportKeepKey))
	ok(w, map[string]int{"keepDays": keep, "fixedDays": s.fixedKeepDays()})
}

// publicBase is the address CarTrak is reached at from outside, or "".
func (s *Server) publicBase() string {
	if s.Cfg != nil {
		if a := s.Cfg.Value(keyAddress); a != "" {
			return "https://" + a
		}
	}
	return ""
}

// makeReportLink: POST /api/bug-reports/link {"days": 7}. One link opens every report as plain text without signing in; the secret
// is in the address. Making a new one ends the old. It expires, and it can be revoked.
func (s *Server) makeReportLink(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Days int `json:"days"`
	}
	if decode(r, &in) != nil || in.Days < 1 || in.Days > 90 {
		in.Days = 7
	}
	tok := shortToken(24)
	if err := s.S.NewReportLink(tok, store.TokenHash(tok), time.Now().AddDate(0, 0, in.Days).Unix()); err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, map[string]any{"path": "/r/" + tok, "url": s.publicBase() + "/r/" + tok, "addressKnown": s.publicBase() != "", "days": in.Days})
}

// revokeReportLink: DELETE /api/bug-reports/link.
func (s *Server) revokeReportLink(w http.ResponseWriter, r *http.Request) {
	if err := s.S.RevokeReportLink(); err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, map[string]bool{"ok": true})
}
