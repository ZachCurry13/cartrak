package server

import (
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	softwareKey      = "software_versions"
	maxSoftwareNotes = 100
)

// softwareNote is one software version the car was on from a day, typed in by the owner: Ford does not send the version, so this
// is the only way to line efficiency and battery trends up with an update.
type softwareNote struct {
	ID      int64  `json:"id"`
	Date    string `json:"date"` // 2026-11-02, the first day on this version
	Version string `json:"version"`
}

func (s *Server) softwareNotes() []softwareNote {
	out := []softwareNote{} // never null, so the page can always count them
	_ = json.Unmarshal([]byte(s.S.GetKV(softwareKey)), &out)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Date > out[j].Date }) // newest first
	return out
}

// listSoftware: GET /api/software
func (s *Server) listSoftware(w http.ResponseWriter, r *http.Request) {
	ok(w, map[string]any{"notes": s.softwareNotes()})
}

// addSoftware: POST /api/software {"date":"2026-11-02","version":"..."}. Admin only.
func (s *Server) addSoftware(w http.ResponseWriter, r *http.Request) {
	var in softwareNote
	if decode(r, &in) != nil {
		fail(w, 400, "give the date and the version")
		return
	}
	in.Version = strings.TrimSpace(in.Version)
	if in.Version == "" || len(in.Version) > 40 {
		fail(w, 400, "type the version as the car shows it, up to 40 characters")
		return
	}
	d, err := time.Parse("2006-01-02", in.Date)
	if err != nil || d.After(time.Now().AddDate(0, 0, 1)) {
		fail(w, 400, "give the first day on this version, not in the future")
		return
	}
	all := s.softwareNotes()
	if len(all) >= maxSoftwareNotes {
		fail(w, 400, "that is a lot of versions; remove some first")
		return
	}
	in.ID = 1
	for _, n := range all {
		if n.ID >= in.ID {
			in.ID = n.ID + 1
		}
	}
	b, _ := json.Marshal(append(all, in))
	if s.S.SetKV(softwareKey, string(b)) != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, map[string]any{"notes": s.softwareNotes()})
}

// deleteSoftware: DELETE /api/software/{id}. Admin only.
func (s *Server) deleteSoftware(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	keep := []softwareNote{}
	for _, n := range s.softwareNotes() {
		if n.ID != id {
			keep = append(keep, n)
		}
	}
	b, _ := json.Marshal(keep)
	if s.S.SetKV(softwareKey, string(b)) != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, map[string]any{"notes": s.softwareNotes()})
}
