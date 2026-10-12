package server

import (
	"context"
	"net/http"

	"github.com/ZachCurry13/cartrak"
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
	})
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
	ok(w, map[string]any{
		"dismissed":  s.S.GetKV("onboarding_dismissed") == "1",
		"linked":     st.Linked,
		"canLink":    s.Link != nil,
		"hasReading": latest != nil,
		"rules":      len(rules),
		"users":      s.S.CountUsers(),
		"provider":   st.Provider,
	})
}

func (s *Server) dismissOnboarding(w http.ResponseWriter, r *http.Request) {
	_ = s.S.SetKV("onboarding_dismissed", "1")
	ok(w, "dismissed")
}
