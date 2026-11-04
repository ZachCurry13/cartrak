package server

import (
	"net/http"
	"strconv"

	"github.com/ZachCurry13/cartrak/internal/store"
)

// servePageCopy writes a report's page copy. It is only the page's markup and styles: scripts were left out when it was saved, and
// the policy below stops it from running anything or loading anything from anywhere.
func servePageCopy(w http.ResponseWriter, r *http.Request, rep *store.BugReport) {
	if rep == nil || rep.HTML == "" {
		http.NotFound(w, r)
		return
	}
	shareHeaders(w)
	h := w.Header()
	h.Set("Content-Type", "text/html; charset=utf-8")
	h.Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; img-src data:; font-src data:; sandbox")
	h.Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write([]byte(rep.HTML))
}

// reportPageCopy: GET /r/{token}/{id}/page. The page as it looked when the report was made.
func (s *Server) reportPageCopy(w http.ResponseWriter, r *http.Request) {
	if !s.reportLinkFor(w, r) {
		return
	}
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	rep, _ := s.S.BugReport(id)
	servePageCopy(w, r, rep)
}

// adminPageCopy: GET /api/bug-reports/{id}/page. The same, for the owner signed in.
func (s *Server) adminPageCopy(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	rep, _ := s.S.BugReport(id)
	servePageCopy(w, r, rep)
}
