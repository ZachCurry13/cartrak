package server

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
)

// RedirectChecker is implemented by providers that can ask Ford whether a redirect address is registered.
type RedirectChecker interface {
	CheckRedirect(ctx context.Context, redirectURI string) (ford.RedirectCheck, error)
}

type detectResult struct {
	Results []ford.RedirectCheck `json:"results"`
	Chosen  string               `json:"chosen"`  // the accepted address now in use ("" if none)
	Changed bool                 `json:"changed"` // CarTrak switched the setting
	Locked  bool                 `json:"locked"`  // set by env/config.yaml, so not changed
	Current string               `json:"current"`
}

// redirectCandidates lists the addresses worth asking Ford about, best first: this
// site's public https address alone, then with /ford/callback, then http://localhost.
// Plain http addresses other than localhost are never worth asking: Ford refuses them.
func (s *Server) redirectCandidates(r *http.Request) []string {
	var origins []string
	add := func(o string) {
		if !strings.HasPrefix(o, "https://") {
			return
		}
		for _, x := range origins {
			if x == o {
				return
			}
		}
		origins = append(origins, o)
	}
	if s.Cfg != nil {
		if a := s.Cfg.Value("CARTRAK_PUBLIC_ADDRESS"); a != "" {
			add("https://" + a)
		}
	}
	add(s.origin(r))
	var out []string
	for _, o := range origins {
		out = append(out, o, o+"/ford/callback")
	}
	return append(out, "http://localhost")
}

// detect asks Ford which candidate redirect addresses are registered for this
// app, and switches CarTrak's redirect URL to the best accepted one (unless the
// current value is already accepted, or is locked by env/config.yaml).
func (s *Server) detect(ctx context.Context, r *http.Request) (detectResult, error) {
	res := detectResult{Current: s.Cfg.Value("FORD_REDIRECT_URL"), Locked: s.Cfg.Locked("FORD_REDIRECT_URL")}
	for _, c := range s.redirectCandidates(r) {
		chk, err := s.Redirects.CheckRedirect(ctx, c)
		if err != nil {
			return res, err
		}
		res.Results = append(res.Results, chk)
		if chk.Accepted && res.Chosen == "" {
			res.Chosen = c
		}
	}
	for _, chk := range res.Results { // keep the current value when Ford accepts it
		if chk.URI == res.Current && chk.Accepted {
			res.Chosen = res.Current
		}
	}
	if res.Chosen != "" && res.Chosen != res.Current && !res.Locked {
		if err := s.Cfg.Set("FORD_REDIRECT_URL", res.Chosen); err == nil {
			res.Changed, res.Current = true, res.Chosen
		}
	}
	return res, nil
}

func (s *Server) detectRedirect(w http.ResponseWriter, r *http.Request) {
	if s.Redirects == nil || s.Cfg == nil {
		fail(w, 404, "this provider cannot check redirect addresses")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()
	res, err := s.detect(ctx, r)
	if err != nil {
		fail(w, 502, "Could not reach Ford to check: "+err.Error())
		return
	}
	ok(w, res)
}

// ensureRedirect runs before sending the browser to Ford: it makes sure the
// redirect address is one Ford accepts, switching to one that is if needed. It
// returns false only when Ford accepts none of the candidates. If Ford cannot be
// reached it lets the sign-in proceed, since it cannot tell.
func (s *Server) ensureRedirect(r *http.Request) bool {
	if s.Redirects == nil || s.Cfg == nil {
		return true
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	if cur := s.Cfg.Value("FORD_REDIRECT_URL"); cur != "" {
		if chk, err := s.Redirects.CheckRedirect(ctx, cur); err != nil || chk.Accepted {
			return true
		}
	}
	res, err := s.detect(ctx, r)
	if err != nil {
		return true
	}
	return res.Chosen != ""
}
