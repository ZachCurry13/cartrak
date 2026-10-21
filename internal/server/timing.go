package server

import (
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// How long each kind of request takes to answer, since CarTrak started, so a slow page can be traced to the request behind it.
// The debug report lists the slowest. Streams, tiles and the app's own files are not timed.

type reqStat struct {
	n          int
	total, max time.Duration
}

var (
	timingMu sync.Mutex
	timings  = map[string]*reqStat{}
	idPath   = regexp.MustCompile(`/\d+`)
)

func timed(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if !strings.HasPrefix(p, "/api/") || strings.HasPrefix(p, "/api/live") || strings.HasPrefix(p, "/api/tiles") {
			h.ServeHTTP(w, r)
			return
		}
		start := time.Now()
		h.ServeHTTP(w, r)
		noteTiming(r.Method+" "+idPath.ReplaceAllString(p, "/{id}"), time.Since(start))
	})
}

func noteTiming(key string, d time.Duration) {
	timingMu.Lock()
	defer timingMu.Unlock()
	s := timings[key]
	if s == nil {
		s = &reqStat{}
		timings[key] = s
	}
	s.n++
	s.total += d
	if d > s.max {
		s.max = d
	}
}

// SlowestRequests is the twelve kinds of request with the longest average answer, as lines for the debug report.
func SlowestRequests() []string {
	timingMu.Lock()
	defer timingMu.Unlock()
	type row struct {
		key string
		avg time.Duration
		s   reqStat
	}
	var rows []row
	for k, s := range timings {
		rows = append(rows, row{k, s.total / time.Duration(s.n), *s})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].avg > rows[j].avg })
	if len(rows) > 12 {
		rows = rows[:12]
	}
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, fmt.Sprintf("%-40s average %5d ms, slowest %5d ms, %d times", r.key, r.avg.Milliseconds(), r.s.max.Milliseconds(), r.s.n))
	}
	return out
}
