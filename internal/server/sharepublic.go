package server

import (
	"io/fs"
	"net/http"

	"github.com/ZachCurry13/cartrak/internal/share"
	"github.com/ZachCurry13/cartrak/internal/store"
	"github.com/ZachCurry13/cartrak/internal/web"
)

// shareHeaders keeps a shared page out of search engines, caches and referrer headers.
func shareHeaders(w http.ResponseWriter) {
	h := w.Header()
	h.Set("Cache-Control", "no-store")
	h.Set("X-Robots-Tag", "noindex, nofollow, noarchive")
	h.Set("Referrer-Policy", "no-referrer")
}

// shareFor finds the valid share for the link's secret. A wrong secret counts against the same lockout as wrong
// passwords, so links cannot be guessed, and every kind of failure looks the same.
func (s *Server) shareFor(w http.ResponseWriter, r *http.Request) *store.Share {
	ip := s.clientIP(r)
	shareHeaders(w)
	if s.lim.blocked(ip) {
		http.Error(w, "too many attempts, wait a minute", http.StatusTooManyRequests)
		return nil
	}
	sh, _ := s.S.ShareByHash(store.TokenHash(r.PathValue("token")))
	if sh == nil {
		s.lim.fail(ip)
		http.NotFound(w, r)
		return nil
	}
	return sh
}

// sharePage is the page a link opens. It is a small page of its own, not the CarTrak app.
func (s *Server) sharePage(w http.ResponseWriter, r *http.Request) {
	sh := s.shareFor(w, r)
	if sh == nil {
		return
	}
	s.S.CountShareView(sh.ID)
	static, _ := fs.Sub(web.Files, "static")
	page, err := fs.ReadFile(static, "share.html")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(s.versionAssets(page))
}

func (s *Server) shareJSON(w http.ResponseWriter, r *http.Request) {
	sh := s.shareFor(w, r)
	if sh == nil {
		return
	}
	ok(w, s.shareData(sh))
}

// shareTile serves map tiles to a shared page, only near the shared route and only if the link includes a route.
func (s *Server) shareTile(w http.ResponseWriter, r *http.Request) {
	sh := s.shareFor(w, r)
	if sh == nil {
		return
	}
	z, x, y, good := tileNumbers(r.PathValue("z"), r.PathValue("x"), r.PathValue("y"))
	if !good || sh.Mode == "stats" || !s.mapsOn() || !share.TileAllowed(sh.BBox, z, x, y) {
		http.NotFound(w, r)
		return
	}
	s.serveTile(w, z, x, y)
}
