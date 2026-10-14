package server

import (
	"context"
	"net/http"
	"runtime"
	"strings"

	"github.com/ZachCurry13/cartrak"
	"github.com/ZachCurry13/cartrak/internal/tunnel"
	"github.com/ZachCurry13/cartrak/internal/updater"
)

// UpdateCheckEnabled is on unless an admin turned it off.
func (s *Server) UpdateCheckEnabled() bool { return s.S.GetKV("update_check") != "off" }

func (s *Server) about(w http.ResponseWriter, r *http.Request) {
	var info updater.Info
	if s.Upd != nil {
		info = s.Upd.Info()
	}
	ok(w, map[string]any{
		"version":       s.Version,
		"updateCheck":   s.UpdateCheckEnabled(),
		"update":        info,
		"provider":      s.P.Status().Provider,
		"releasesUrl":   "https://github.com/ZachCurry13/cartrak/releases",
		"changelogSeen": s.Version,
		"components":    s.components(info),
	})
}

type component struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Latest  string `json:"latest,omitempty"`
	Update  bool   `json:"update"`
	Note    string `json:"note,omitempty"`
}

// components lists the parts CarTrak is built from, and which of them can be updated from the app.
func (s *Server) components(info updater.Info) []component {
	out := []component{{Name: "CarTrak", Version: s.Version, Latest: info.Latest, Update: info.Available, Note: "Updates through your TrueNAS app (Stop, then Start)."}}
	if s.Conn != nil {
		c := s.Conn.Info()
		out = append(out, component{Name: "Cloudflare connector", Version: c.Current, Latest: c.Latest, Update: c.Available, Note: "Can be updated from Remote access."})
	}
	var sq string
	_ = s.S.DB.QueryRow(`SELECT sqlite_version()`).Scan(&sq)
	return append(out, component{Name: "SQLite", Version: sq, Note: "Comes with CarTrak."}, component{Name: "Go", Version: strings.TrimPrefix(runtime.Version(), "go"), Note: "Comes with CarTrak."})
}

func (s *Server) setUpdateCheck(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Enabled bool `json:"enabled"`
	}
	if decode(r, &b) != nil {
		fail(w, 400, "bad request")
		return
	}
	v := "off"
	if b.Enabled {
		v = "on"
	}
	_ = s.S.SetKV("update_check", v)
	if b.Enabled && s.Upd != nil {
		go func() { _ = s.Upd.Check(context.Background()) }()
	}
	ok(w, map[string]bool{"enabled": b.Enabled})
}

func (s *Server) changelog(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	_, _ = w.Write([]byte(cartrak.Changelog))
}

// onboarding reports what the getting-started checklist can know server-side.
func (s *Server) onboarding(w http.ResponseWriter, r *http.Request) {
	st := s.P.Status()
	latest, _ := s.S.LatestSample()
	rules, _ := s.S.ListRules()
	named, _ := s.S.ListNamedPlaces()
	maintenance, _ := s.S.ListMaintenance()
	alerts := 0
	if s.Push != nil {
		if subs, err := s.S.ListPushSubs(0); err == nil {
			alerts = len(subs)
		}
	}
	pr := s.prices()
	ok(w, map[string]any{
		"places":      len(named),
		"maintenance": len(maintenance),
		"alerts":      alerts,
		"priceSet":    pr.Home != nil || pr.Public != nil,
		"backedUp":    s.S.GetKV(kvBackupTaken) != "",
		"dismissed":   s.S.GetKV("onboarding_dismissed") == "1",
		"linked":      st.Linked,
		"canLink":     s.Link != nil,
		"configured":  s.Cfg == nil || s.Cfg.Configured(),
		"remote":      s.Tunnel != nil && s.Tunnel.Status().State == tunnel.Connected,
		"hasReading":  latest != nil,
		"rules":       len(rules),
		"users":       s.S.CountUsers(),
		"provider":    st.Provider,
	})
}

func (s *Server) dismissOnboarding(w http.ResponseWriter, r *http.Request) {
	_ = s.S.SetKV("onboarding_dismissed", "1")
	ok(w, "dismissed")
}
