package server

import (
	"net/http"
	"strings"

	"github.com/ZachCurry13/cartrak/internal/settings"
)

type settingView struct {
	settings.Def
	Value  string `json:"value"` // always empty for secrets
	IsSet  bool   `json:"isSet"`
	Source string `json:"source"` // env | config.yaml | admin | default
	Locked bool   `json:"locked"` // set by env/config.yaml: edit it there
}

func (s *Server) settingsView() []settingView {
	out := make([]settingView, 0, len(settings.Defs))
	for _, d := range settings.Defs {
		v, src := s.Cfg.Get(d.Key)
		sv := settingView{Def: d, IsSet: v != "", Source: src, Locked: s.Cfg.Locked(d.Key)}
		if !d.Secret {
			sv.Value = v
		}
		out = append(out, sv)
	}
	return out
}

func (s *Server) getSettings(w http.ResponseWriter, r *http.Request) {
	if s.Cfg == nil {
		fail(w, 503, "settings unavailable")
		return
	}
	ok(w, map[string]any{"settings": s.settingsView(), "suggestedRedirect": originOf(r) + "/ford/callback"})
}

// originOf rebuilds the address the browser used, e.g. https://cartrak.example.com.
func originOf(r *http.Request) string {
	scheme := "http"
	if secure(r) {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

// saveSettings stores admin-panel values. Locked keys are refused, a blank
// secret means "keep the current one", and "clear" removes a stored value.
func (s *Server) saveSettings(w http.ResponseWriter, r *http.Request) {
	if s.Cfg == nil {
		fail(w, 503, "settings unavailable")
		return
	}
	var b struct {
		Values map[string]string `json:"values"`
		Clear  []string          `json:"clear"`
	}
	if decode(r, &b) != nil {
		fail(w, 400, "bad request")
		return
	}
	var locked []string
	for k := range b.Values {
		d, known := settings.Known(k)
		if !known {
			fail(w, 400, "unknown setting "+k)
			return
		}
		if s.Cfg.Locked(k) {
			locked = append(locked, d.Label)
		}
	}
	for _, k := range b.Clear {
		if d, known := settings.Known(k); !known {
			fail(w, 400, "unknown setting "+k)
			return
		} else if s.Cfg.Locked(k) {
			locked = append(locked, d.Label)
		}
	}
	if len(locked) > 0 {
		fail(w, 409, "set by an environment variable or config.yaml, change it there: "+strings.Join(locked, ", "))
		return
	}
	for k, v := range b.Values {
		d, _ := settings.Known(k)
		if d.Secret && strings.TrimSpace(v) == "" {
			continue
		}
		if err := settings.Valid(k, v); err != nil {
			fail(w, 400, d.Label+": "+err.Error())
			return
		}
	}
	for k, v := range b.Values {
		if d, _ := settings.Known(k); d.Secret && strings.TrimSpace(v) == "" {
			continue
		}
		if err := s.Cfg.Set(k, v); err != nil {
			fail(w, 500, "could not save")
			return
		}
	}
	for _, k := range b.Clear {
		_ = s.Cfg.Set(k, "")
	}
	ok(w, map[string]any{"settings": s.settingsView(), "suggestedRedirect": originOf(r) + "/ford/callback"})
}
