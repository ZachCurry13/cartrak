package server

import (
	"bytes"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// briefTTL is how long the answer to a slow, read-only call is reused. Home asks for a dozen of these at once, each waiting its turn
// on the one database connection, so a page that took seconds is the sum of them. A new reading every half minute or so is the
// most these figures change by themselves, and anything a person changes (any write) throws the kept answers away at once.
const briefTTL = 20 * time.Second

// briefDisabled turns the reuse off. Tests that change the database directly and read it back at once need to see the change.
var briefDisabled bool

type briefCache struct {
	mu  sync.Mutex
	m   map[string]briefEntry
	gen atomic.Int64 // counts writes: an answer kept before the latest one is no longer used
}

type briefEntry struct {
	at     time.Time
	gen    int64
	status int
	header http.Header
	body   []byte
}

type recorder struct {
	status int
	header http.Header
	body   bytes.Buffer
}

func (r *recorder) Header() http.Header         { return r.header }
func (r *recorder) WriteHeader(code int)        { r.status = code }
func (r *recorder) Write(b []byte) (int, error) { return r.body.Write(b) }

// briefly reuses the answer to a GET for briefTTL. Only a successful answer is kept, and only until the next write of any kind.
func (s *Server) briefly(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || briefDisabled {
			h(w, r)
			return
		}
		key, gen := r.URL.RequestURI(), s.brief.gen.Load()
		s.brief.mu.Lock()
		e, ok := s.brief.m[key]
		s.brief.mu.Unlock()
		if !ok || e.gen != gen || time.Since(e.at) > briefTTL {
			rec := &recorder{status: http.StatusOK, header: http.Header{}}
			h(rec, r)
			e = briefEntry{at: time.Now(), gen: gen, status: rec.status, header: rec.header, body: rec.body.Bytes()}
			if e.status == http.StatusOK {
				s.brief.mu.Lock()
				if s.brief.m == nil || len(s.brief.m) > 200 {
					s.brief.m = map[string]briefEntry{}
				}
				s.brief.m[key] = e
				s.brief.mu.Unlock()
			}
		}
		for k, v := range e.header {
			w.Header()[k] = v
		}
		w.WriteHeader(e.status)
		w.Write(e.body)
	}
}

// writes counts every call that is not a plain read, so briefly never serves an answer from before a change.
func (s *Server) writes(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			s.brief.gen.Add(1)
			defer s.brief.gen.Add(1) // and again when it is done, in case a read slipped in between
		}
		next.ServeHTTP(w, r)
	})
}
