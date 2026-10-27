package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/ZachCurry13/cartrak/internal/settings"
)

// The Field check answers (More, Developer) are kept on the server as well as on the phone, so the Debug report and a tracking
// export carry them and nothing has to be pasted.
const (
	kvFieldCheck = "field_check"
	maxFieldBody = 24 << 10
)

func (s *Server) fieldCheckGet(w http.ResponseWriter, r *http.Request) {
	var v any
	if raw := s.S.GetKV(kvFieldCheck); raw != "" {
		_ = json.Unmarshal([]byte(raw), &v)
	}
	ok(w, map[string]any{"state": v})
}

// fieldCheckPut: PUT /api/fieldcheck {"state": {...}}. The page's own answers, stored as they are.
func (s *Server) fieldCheckPut(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxFieldBody+1))
	if err != nil || len(body) > maxFieldBody {
		fail(w, 400, "too large")
		return
	}
	var in struct {
		State json.RawMessage `json:"state"`
	}
	if json.Unmarshal(body, &in) != nil || len(in.State) == 0 {
		fail(w, 400, "bad request")
		return
	}
	_ = s.S.SetKV(kvFieldCheck, string(in.State))
	ok(w, "saved")
}

type fieldAnswer struct {
	S    string `json:"s"`
	Note string `json:"note"`
}

// fieldCheckText is the saved Field check answers as lines for the Debug report, or "" when there are none.
func (s *Server) fieldCheckText() string {
	var st struct {
		At     int64                  `json:"at"`
		Checks map[string]fieldAnswer `json:"checks"`
		Auto   [][2]string            `json:"auto"`
		Titles map[string]string      `json:"titles"`
	}
	if json.Unmarshal([]byte(s.S.GetKV(kvFieldCheck)), &st) != nil || (len(st.Checks) == 0 && len(st.Auto) == 0) {
		return ""
	}
	var b strings.Builder
	b.WriteString("\nFIELD CHECK (the answers saved on the Field check page")
	if st.At > 0 {
		fmt.Fprintf(&b, ", last changed %s", time.Unix(st.At, 0).In(s.defaultLoc()).Format("01-02 15:04"))
	}
	b.WriteString(")\n")
	for _, a := range st.Auto {
		fmt.Fprintf(&b, "    checked for you | %s: %s\n", a[0], a[1])
	}
	ids := make([]string, 0, len(st.Checks))
	for id := range st.Checks {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		c := st.Checks[id]
		name := st.Titles[id]
		if name == "" {
			name = id
		}
		label := map[string]string{"ok": "works", "bad": "PROBLEM", "skip": "skipped"}[c.S]
		if label == "" {
			label = "not done"
		}
		note := ""
		if c.Note != "" {
			note = " | " + c.Note
		}
		fmt.Fprintf(&b, "    %s: %s%s\n", name, label, note)
	}
	return b.String()
}

// logSettingChange notes which settings were changed (never their values) so a tracking export and the Debug report show it.
func (s *Server) logSettingChange(values map[string]string, cleared []string) {
	var names []string
	for k, v := range values {
		if d, known := settings.Known(k); known && !(d.Secret && strings.TrimSpace(v) == "") {
			names = append(names, d.Label)
		}
	}
	for _, k := range cleared {
		if d, known := settings.Known(k); known {
			names = append(names, d.Label+" (cleared)")
		}
	}
	if len(names) > 0 {
		sort.Strings(names)
		s.S.LogDecision("setting", "changed: "+strings.Join(names, ", "))
	}
}
