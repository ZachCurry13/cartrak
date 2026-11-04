package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

const (
	reportKeepKey  = "report_keep_days"
	reportMaxKept  = 200
	reportBodyMax  = 3 << 20
	reportDebugMax = 1 << 20
	reportPageMax  = 1 << 20 // what one link answers with
)

// pruneReports forgets reports older than the keep setting (0 keeps them all).
func (s *Server) pruneReports() {
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
	ok(w, map[string]any{"reports": l, "keepDays": keep, "link": s.S.CurrentReportLink(), "addressKnown": s.publicBase() != ""})
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
	if len(in.Note) > 4000 || len(in.Page) > 200 || len(in.Device) > 1000 || len(in.Text) > 60000 || len(in.Layout) > 30000 || len(in.Debug) > reportDebugMax {
		fail(w, 400, "part of that report is too long")
		return
	}
	if all, _ := s.S.BugReports(false); len(all) >= reportMaxKept {
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
		Days int `json:"days"`
	}
	if decode(r, &in) != nil || (in.Days != 0 && in.Days != 30 && in.Days != 60 && in.Days != 90) {
		fail(w, 400, "keep reports for 0 (until cleared), 30, 60 or 90 days")
		return
	}
	_ = s.S.SetKV(reportKeepKey, strconv.Itoa(in.Days))
	s.pruneReports()
	ok(w, map[string]int{"keepDays": in.Days})
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
	if err := s.S.NewReportLink(store.TokenHash(tok), time.Now().AddDate(0, 0, in.Days).Unix()); err != nil {
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

// reportText is one report as plain text.
func reportText(r *store.BugReport, debugMax int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Report %d, %s, app %s\n\n", r.ID, time.Unix(r.TS, 0).Format("2006-01-02 15:04"), r.Version)
	fmt.Fprintf(&b, "Note: %s\n\nPage: %s\nDevice: %s\n\n", strings.TrimSpace(r.Note), r.Page, r.Device)
	if r.Layout != "" {
		fmt.Fprintf(&b, "## Layout facts\n%s\n\n", r.Layout)
	}
	if r.Text != "" {
		fmt.Fprintf(&b, "## What the page said\n%s\n\n", r.Text)
	}
	if r.Debug != "" {
		d := r.Debug
		if len(d) > debugMax {
			d = d[:debugMax] + "\n[cut]"
		}
		fmt.Fprintf(&b, "## Debug report\n%s\n", d)
	}
	return b.String()
}

// reportLinkFor checks the link's secret, with the same lockout as wrong passwords. Every kind of failure looks the same.
func (s *Server) reportLinkFor(w http.ResponseWriter, r *http.Request) bool {
	ip := s.clientIP(r)
	shareHeaders(w)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if s.lim.blocked(ip) {
		http.Error(w, "too many attempts, wait a minute", http.StatusTooManyRequests)
		return false
	}
	if !s.S.ReportLinkOK(store.TokenHash(r.PathValue("token")), time.Now().Unix()) {
		s.lim.fail(ip)
		http.NotFound(w, r)
		return false
	}
	return true
}

// reportsPage: GET /r/{token}. Every report as plain text, newest first.
func (s *Server) reportsPage(w http.ResponseWriter, r *http.Request) {
	if !s.reportLinkFor(w, r) {
		return
	}
	l, _ := s.S.BugReports(true)
	var b strings.Builder
	fmt.Fprintf(&b, "CarTrak bug reports: %d, newest first. Each is also at %s/<number>.\n\n", len(l), strings.TrimSuffix(r.URL.Path, "/"))
	for i := range l {
		t := reportText(&l[i], 60000)
		if b.Len()+len(t) > reportPageMax {
			fmt.Fprintf(&b, "[%d older reports left out: open them one at a time by number]\n", len(l)-i)
			break
		}
		b.WriteString(t + "\n---\n\n")
	}
	_, _ = w.Write([]byte(b.String()))
}

// reportPage: GET /r/{token}/{id}. One report as plain text, with its whole debug report.
func (s *Server) reportPage(w http.ResponseWriter, r *http.Request) {
	if !s.reportLinkFor(w, r) {
		return
	}
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	rep, _ := s.S.BugReport(id)
	if rep == nil {
		http.NotFound(w, r)
		return
	}
	_, _ = w.Write([]byte(reportText(rep, reportDebugMax)))
}
