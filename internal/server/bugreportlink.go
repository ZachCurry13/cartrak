package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

// What a bug report says in words, and the pages behind the secret link the owner gives out to have reports read
// (see bugreports.go for the owner's own list, and docs/bug-report-links.md).

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
