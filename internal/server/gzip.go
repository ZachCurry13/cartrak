package server

import (
	"compress/gzip"
	"net/http"
	"strings"
	"sync"
)

var gzPool = sync.Pool{New: func() any { return gzip.NewWriter(nil) }}

// gzipHandler compresses text responses (JSON, HTML, CSS, JS) for clients that
// accept it. The live event stream is skipped: it must not be buffered.
func gzipHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") || r.URL.Path == "/api/live" {
			next.ServeHTTP(w, r)
			return
		}
		gw := &gzipWriter{ResponseWriter: w}
		defer gw.close()
		w.Header().Add("Vary", "Accept-Encoding")
		next.ServeHTTP(gw, r)
	})
}

type gzipWriter struct {
	http.ResponseWriter
	zw      *gzip.Writer
	decided bool
}

func compressible(ct string) bool {
	for _, p := range []string{"application/json", "text/", "application/javascript", "image/svg+xml", "application/manifest+json"} {
		if strings.HasPrefix(ct, p) {
			return true
		}
	}
	return false
}

func (g *gzipWriter) decide(status int) {
	if g.decided {
		return
	}
	g.decided = true
	h := g.Header()
	if status == http.StatusNotModified || status == http.StatusNoContent || h.Get("Content-Encoding") != "" ||
		!compressible(h.Get("Content-Type")) {
		return
	}
	h.Set("Content-Encoding", "gzip")
	h.Del("Content-Length")
	g.zw = gzPool.Get().(*gzip.Writer)
	g.zw.Reset(g.ResponseWriter)
}

func (g *gzipWriter) WriteHeader(status int) {
	g.decide(status)
	g.ResponseWriter.WriteHeader(status)
}

func (g *gzipWriter) Write(b []byte) (int, error) {
	if !g.decided {
		if g.Header().Get("Content-Type") == "" {
			g.Header().Set("Content-Type", http.DetectContentType(b))
		}
		g.decide(http.StatusOK)
	}
	if g.zw != nil {
		return g.zw.Write(b)
	}
	return g.ResponseWriter.Write(b)
}

func (g *gzipWriter) Flush() {
	if g.zw != nil {
		_ = g.zw.Flush()
	}
	if f, ok := g.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (g *gzipWriter) close() {
	if g.zw != nil {
		_ = g.zw.Close()
		gzPool.Put(g.zw)
	}
}
