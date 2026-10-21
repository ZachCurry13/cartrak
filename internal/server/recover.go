package server

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

// A request that hits a bug answers with an error the app can show, instead of a dropped connection (which a Cloudflare tunnel
// reports as "502"). The last few are kept, with where in the code it happened, for the debug report.

type serverError struct {
	At    time.Time
	Path  string
	What  string
	Where string // the first lines of the stack in CarTrak's own code
}

var (
	errMu  sync.Mutex
	errLog []serverError
)

func noteServerError(e serverError) {
	errMu.Lock()
	defer errMu.Unlock()
	errLog = append(errLog, e)
	if len(errLog) > 10 {
		errLog = errLog[len(errLog)-10:]
	}
}

// RecentServerErrors is the last few, oldest first, as lines for the debug report.
func RecentServerErrors() []string {
	errMu.Lock()
	defer errMu.Unlock()
	out := make([]string, 0, len(errLog))
	for _, e := range errLog {
		out = append(out, fmt.Sprintf("%s %s: %s (%s)", e.At.UTC().Format("2006-01-02 15:04:05"), e.Path, e.What, e.Where))
	}
	return out
}

// ownFrames picks the stack lines in CarTrak's own packages, the useful part.
func ownFrames(stack string) string {
	var keep []string
	for _, l := range strings.Split(stack, "\n") {
		if strings.Contains(l, "cartrak/internal/") && !strings.Contains(l, "recover.go") {
			keep = append(keep, strings.TrimSpace(l))
		}
		if len(keep) == 4 {
			break
		}
	}
	return strings.Join(keep, " < ")
}

func recoverer(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				if v == http.ErrAbortHandler {
					panic(v)
				}
				stack := string(debug.Stack())
				log.Printf("error on %s %s: %v\n%s", r.Method, r.URL.Path, v, stack)
				noteServerError(serverError{At: time.Now(), Path: r.Method + " " + r.URL.Path, What: fmt.Sprint(v), Where: ownFrames(stack)})
				fail(w, 500, "CarTrak hit a problem with that. It has been noted for the debug report; please send one.")
			}
		}()
		h.ServeHTTP(w, r)
	})
}
