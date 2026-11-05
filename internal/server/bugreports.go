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
	reportBodyMax  = 5 << 20
	reportHTMLMax  = 2 << 20
	reportDebugMax = 1 << 20
	reportPageMax  = 1 << 20 // what one link answers with
)

// pruneReports forgets reports older than the keep setting (0 keeps them all).
func (s *Server) pruneReports() {
	_, _ = s.S.DeleteFixedBugReports(time.Now().Add(-fixedKeep).Unix()) // a fixed report goes a week after it was fixed
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
	ok(w, map[string]any{"reports": l, "keepDays": keep, "link": link, "linkUrl": url, "addressKnown": s.publicBase() != ""})
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

const reportTextMax = 6000 // what one report says of the page's words; the page copy has the rest

func shortText(s string, n int) string {
	if len(s) > n {
		return s[:n] + " [cut]"
	}
	return s
}

// reportText is one report as plain text, without its debug report (that is at ./<number>/debug, so a report stays short).
func reportText(r *store.BugReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Report %d, %s, app %s\n\n", r.ID, time.Unix(r.TS, 0).Format("2006-01-02 15:04"), r.Version)
	fmt.Fprintf(&b, "Note: %s\n\nPage: %s\nDevice: %s\n", strings.TrimSpace(r.Note), r.Page, r.Device)
	if r.HasPage {
		fmt.Fprintf(&b, "The page as it looked (open it at the window size above): ./%d/page\n", r.ID)
	}
	if r.HasDebug {
		fmt.Fprintf(&b, "The debug report, only if the problem is about the numbers: ./%d/debug\n", r.ID)
	}
	b.WriteString("\n")
	if r.Layout != "" {
		fmt.Fprintf(&b, "## Layout facts\n%s\n\n", r.Layout)
	}
	if r.Text != "" {
		fmt.Fprintf(&b, "## What the page said\n%s\n", shortText(r.Text, reportTextMax))
	}
	return b.String()
}

// reportBrief is a report in two or three lines, for the index.
func reportBrief(r *store.BugReport, base string) string {
	device := r.Device
	if i := strings.Index(device, ";"); i > 0 {
		device = device[:i]
	}
	layout := ""
	if lines := strings.Split(strings.TrimSpace(r.Layout), "\n"); r.Layout != "" {
		if len(lines) > 1 {
			lines = lines[1:] // the first line is only the window size, which the device line already gives
		}
		layout = " Layout: " + shortText(strings.Join(lines, " "), 160)
	}
	more := fmt.Sprintf("full: %s/%d", base, r.ID)
	if r.HasPage {
		more += fmt.Sprintf(" · page copy: %s/%d/page", base, r.ID)
	}
	return fmt.Sprintf("#%d %s · %s · %s\n  Note: %s\n %s\n  %s\n", r.ID, time.Unix(r.TS, 0).Format("2006-01-02 15:04"), r.Page, device, shortText(strings.TrimSpace(r.Note), 400), layout, more)
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

// reportsPage: GET /r/{token}. A short index of the reports, newest first: the note, page, device and layout facts of each and where
// to open the rest. It stays small however many reports there are.
func (s *Server) reportsPage(w http.ResponseWriter, r *http.Request) {
	if !s.reportLinkFor(w, r) {
		return
	}
	all, _ := s.S.BugReportBriefs()
	var l []store.BugReport // the open ones: a fixed report is done with
	for _, x := range all {
		if x.Fixed == 0 {
			l = append(l, x)
		}
	}
	base := strings.TrimSuffix(r.URL.Path, "/")
	var b strings.Builder
	fmt.Fprintf(&b, "CarTrak bug reports: %d open (%d fixed, hidden), newest first. Open one with %s/<number>; the page as it looked is %s/<number>/page. When one is dealt with, POST to %s/<number>/fixed to mark it fixed.\n\n", len(l), len(all)-len(l), base, base, base)
	for i := range l {
		t := reportBrief(&l[i], base)
		if b.Len()+len(t) > reportPageMax {
			fmt.Fprintf(&b, "[%d older reports left out: open them by number]\n", len(l)-i)
			break
		}
		b.WriteString(t + "\n")
	}
	_, _ = w.Write([]byte(b.String()))
}

// reportDebug: GET /r/{token}/{id}/debug. The report's debug report on its own, for problems with the numbers.
func (s *Server) reportDebug(w http.ResponseWriter, r *http.Request) {
	if !s.reportLinkFor(w, r) {
		return
	}
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	rep, _ := s.S.BugReport(id)
	if rep == nil || rep.Debug == "" {
		http.NotFound(w, r)
		return
	}
	_, _ = w.Write([]byte(rep.Debug))
}

// reportPage: GET /r/{token}/{id}. One report as plain text, without its debug report.
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
	_, _ = w.Write([]byte(reportText(rep)))
}
