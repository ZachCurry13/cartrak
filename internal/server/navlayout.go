package server

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// A person's own bottom bar and More tiles: which pages, in which order, and whether they follow use. Kept per person, so they
// follow them to every device. GET and PUT /api/me/layout; the page keys are the app's, checked only for shape here.

type navLayout struct {
	Bar   []string `json:"bar"`
	Tiles []string `json:"tiles"`
	Auto  bool     `json:"auto"`
}

func layoutKey(uid int64) string { return fmt.Sprintf("layout:%d", uid) }

func plainKeys(keys []string, most int) bool {
	if len(keys) > most {
		return false
	}
	for _, k := range keys {
		if len(k) == 0 || len(k) > 20 {
			return false
		}
		for _, c := range k {
			if c < 'a' || c > 'z' {
				return false
			}
		}
	}
	return true
}

func (s *Server) getLayout(w http.ResponseWriter, r *http.Request) {
	var l navLayout
	if raw := s.core().GetKV(layoutKey(userFrom(r).ID)); raw != "" && json.Unmarshal([]byte(raw), &l) == nil {
		ok(w, l)
		return
	}
	ok(w, map[string]any{}) // nothing chosen: the app's default
}

func (s *Server) putLayout(w http.ResponseWriter, r *http.Request) {
	var l navLayout
	if decode(r, &l) != nil || !plainKeys(l.Bar, 6) || !plainKeys(l.Tiles, 12) || len(l.Bar) == 0 {
		fail(w, 400, "choose one to six pages for the bar and up to twelve tiles")
		return
	}
	b, _ := json.Marshal(l)
	_ = s.core().SetKV(layoutKey(userFrom(r).ID), string(b))
	ok(w, l)
}
