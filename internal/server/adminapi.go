package server

import (
	"context"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
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

// settingsView lists the settings of one page: group "" is the general settings, "charging" the charging ones.
func (s *Server) settingsView(group string) []settingView {
	out := make([]settingView, 0, len(settings.Defs))
	for _, d := range settings.Defs {
		if settings.GroupOf(d) != group { // remote access and charging have pages of their own
			continue
		}
		v, src := s.Cfg.Get(d.Key)
		sv := settingView{Def: d, IsSet: v != "", Source: src, Locked: s.Cfg.Locked(d.Key)}
		sv.Group = settings.GroupOf(d)
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
	ok(w, map[string]any{"settings": s.settingsView(r.URL.Query().Get("group")), "suggestedRedirect": s.origin(r) + "/ford/callback"})
}

// originOf rebuilds the address the browser used, e.g. https://cartrak.example.com.
func (s *Server) origin(r *http.Request) string {
	if s.Cfg != nil {
		if a := s.Cfg.Value("CARTRAK_PUBLIC_ADDRESS"); a != "" { // remote access is set up: that is the address Ford should come back to
			return "https://" + a
		}
	}
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
		if d.Group == "remote" && !siteAdmin(r) { // remote access belongs to the whole installation, not to one car
			fail(w, 403, "only an administrator of the whole installation can change remote access")
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
		} else if d.Group == "remote" && !siteAdmin(r) {
			fail(w, 403, "only an administrator of the whole installation can change remote access")
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
	s.logSettingChange(b.Values, b.Clear)
	ok(w, map[string]any{"settings": s.settingsView(r.URL.Query().Get("group")), "suggestedRedirect": s.origin(r) + "/ford/callback"})
}

// Prober is implemented by providers that can call every endpoint once and
// return the redacted answers (location and VIN removed) for debugging.
type Prober interface {
	Probe(ctx context.Context) map[string]ford.ProbeResult
}

// fordProbe lets an admin see exactly what Ford returns, safe to copy and share.
func (s *Server) fordProbe(w http.ResponseWriter, r *http.Request) {
	if s.Prober == nil {
		fail(w, 404, "this provider has no diagnostics")
		return
	}
	if !s.P.Status().Linked {
		fail(w, 409, "link your Ford account first")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	ok(w, map[string]any{
		"results": s.Prober.Probe(ctx),
		"note":    "Location and VIN values are replaced with <redacted>. Tokens are never included.",
	})
}
