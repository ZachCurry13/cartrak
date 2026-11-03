package server

import (
	"encoding/base64"
	"net/http"
	"strings"
	"time"

	"github.com/ZachCurry13/cartrak/internal/bugreport"
)

const (
	reportBody       = 14 << 20 // the whole request
	reportShotLimit  = 8 << 20  // the screenshot once decoded
	reportDebugLimit = 2 << 20
)

type reportIn struct {
	Note       string `json:"note"`
	Page       string `json:"page"`
	Device     string `json:"device"`
	Text       string `json:"text"`
	Debug      string `json:"debug"`
	Screenshot string `json:"screenshot"` // a data:image/png;base64,... address
}

// reportStatus says whether bug reports are set up, so the page can offer them or explain what is missing. GET /api/bug-report/status
func (s *Server) reportStatus(w http.ResponseWriter, r *http.Request) {
	ok(w, map[string]any{"configured": s.Cfg != nil && s.Cfg.ReportRepo() != "" && s.Cfg.ReportToken() != ""})
}

// sendReport sends a bug report to the private repository set up in Settings. It goes off the box, so it is for administrators.
// POST /api/bug-report
func (s *Server) sendReport(w http.ResponseWriter, r *http.Request) {
	if s.Cfg == nil || s.Cfg.ReportRepo() == "" || s.Cfg.ReportToken() == "" {
		fail(w, 409, "set the bug report repository and token in More, Settings first")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, reportBody)
	var in reportIn
	if decode(r, &in) != nil {
		fail(w, 413, "that report is too large to send")
		return
	}
	if len(in.Note) > 4000 || len(in.Page) > 200 || len(in.Device) > 1000 || len(in.Text) > 60000 || len(in.Debug) > reportDebugLimit {
		fail(w, 400, "part of that report is too long")
		return
	}
	var shot []byte
	if in.Screenshot != "" {
		b, found := strings.CutPrefix(in.Screenshot, "data:image/png;base64,")
		if !found {
			fail(w, 400, "the screenshot must be a PNG")
			return
		}
		data, err := base64.StdEncoding.DecodeString(b)
		if err != nil || len(data) < 8 || string(data[1:4]) != "PNG" || len(data) > reportShotLimit {
			fail(w, 400, "the screenshot could not be used")
			return
		}
		shot = data
	}
	c := &bugreport.Client{Repo: s.Cfg.ReportRepo(), Token: s.Cfg.ReportToken(), API: s.ReportAPI}
	res, err := c.Send(r.Context(), bugreport.Report{Version: s.Version, Note: in.Note, Page: in.Page, Device: in.Device, PageText: in.Text, Debug: in.Debug, Screenshot: shot}, time.Now())
	if err != nil {
		fail(w, upstreamFailed, err.Error())
		return
	}
	s.S.LogDecision("bug_report", "a bug report was sent to the private repository: "+res.ID)
	ok(w, res)
}
