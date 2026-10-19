package server

import (
	"io/fs"
	"net/http"
	"regexp"
	"strings"
)

// The app's files are served under a path that carries the version (/v0.25.0/js/app.js) and the pages that point at
// them are never cached. A phone, a browser or a CDN such as Cloudflare in front of the server can then keep the
// files as long as it likes: a new version has new addresses, so an old copy can never be served for it.

var versionRe = regexp.MustCompile(`[^A-Za-z0-9._-]`)

// assetVersion is the version as it appears in an address.
func (s *Server) assetVersion() string {
	v := versionRe.ReplaceAllString(s.Version, "")
	if v == "" {
		return "dev"
	}
	return v
}

// versionAssets points a page's own scripts and styles at the versioned addresses.
func (s *Server) versionAssets(page []byte) []byte {
	v := "/v" + s.assetVersion()
	r := strings.NewReplacer(`href="/css/`, `href="`+v+`/css/`, `src="/js/`, `src="`+v+`/js/`)
	return []byte(r.Replace(string(page)))
}

// staticHandler serves the embedded app. static is the folder of files.
func (s *Server) staticHandler(static fs.FS) http.Handler {
	files := http.FileServer(http.FS(static))
	ver := s.assetVersion()
	prefix := "/v" + ver + "/"
	oldPrefix := regexp.MustCompile(`^/v[A-Za-z0-9._-]+/`)
	templated := func(w http.ResponseWriter, r *http.Request, name, ctype string, edit func([]byte) []byte) {
		b, err := fs.ReadFile(static, name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", ctype)
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write(edit(b))
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		switch {
		case p == "/" || p == "/index.html":
			templated(w, r, "index.html", "text/html; charset=utf-8", s.versionAssets)
		case p == "/sw.js":
			w.Header().Set("Service-Worker-Allowed", "/")
			templated(w, r, "sw.js", "text/javascript; charset=utf-8", func(b []byte) []byte {
				return []byte(strings.ReplaceAll(string(b), "__V__", ver))
			})
		case strings.HasPrefix(p, prefix) || oldPrefix.MatchString(p):
			// Versioned files never change, so they may be kept for a year. A page that still points at an older
			// version's address (an old copy of the page) gets the current file, uncached.
			r2 := r.Clone(r.Context())
			r2.URL.Path = "/" + strings.TrimPrefix(oldPrefix.ReplaceAllString(p, ""), "/")
			if strings.HasPrefix(p, prefix) && ver != "dev" {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			} else {
				w.Header().Set("Cache-Control", "no-cache")
			}
			files.ServeHTTP(w, r2)
		default:
			w.Header().Set("Cache-Control", "no-cache")
			files.ServeHTTP(w, r)
		}
	})
}
