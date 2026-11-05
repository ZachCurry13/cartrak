package server

import (
	"net/http"
	"strconv"
	"time"
)

const fixedKeepKey = "report_fixed_keep_days"

// fixedKeepDays is how many days a report marked fixed is kept before it is cleared: 0 clears it at once, 7 is the default.
func (s *Server) fixedKeepDays() int {
	v := s.S.GetKV(fixedKeepKey)
	if n, err := strconv.Atoi(v); v != "" && err == nil && n >= 0 && n <= 90 {
		return n
	}
	return 7
}

// markFixed: PUT /api/bug-reports/{id}/fixed {"fixed": true}. The owner marks a report fixed, or open again.
func (s *Server) markFixed(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	var in struct {
		Fixed bool `json:"fixed"`
	}
	if id <= 0 || decode(r, &in) != nil {
		fail(w, 400, "say which report")
		return
	}
	at := int64(0)
	if in.Fixed {
		at = time.Now().Unix()
	}
	if err := s.S.SetBugReportFixed(id, at); err != nil {
		fail(w, 500, "db error")
		return
	}
	s.pruneReports() // with "clear at once" a report marked fixed is gone now
	ok(w, map[string]bool{"fixed": in.Fixed})
}

// clearFixed: POST /api/bug-reports/clear-fixed. Removes every report marked fixed.
func (s *Server) clearFixed(w http.ResponseWriter, r *http.Request) {
	n, err := s.S.DeleteFixedBugReports(0)
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, map[string]int64{"removed": n})
}

// linkMarkFixed: POST /r/{token}/{id}/fixed. Whoever holds the report link can mark a report fixed once the problem is dealt with. That
// is all the link can change: it cannot delete, and the owner can open the report again.
func (s *Server) linkMarkFixed(w http.ResponseWriter, r *http.Request) {
	if !s.reportLinkFor(w, r) {
		return
	}
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if rep, _ := s.S.BugReport(id); rep == nil {
		http.NotFound(w, r)
		return
	}
	if err := s.S.SetBugReportFixed(id, time.Now().Unix()); err != nil {
		http.Error(w, "could not save", http.StatusInternalServerError)
		return
	}
	if s.fixedKeepDays() == 0 {
		s.pruneReports()
		_, _ = w.Write([]byte("marked fixed and cleared\n"))
		return
	}
	_, _ = w.Write([]byte("marked fixed\n"))
}
