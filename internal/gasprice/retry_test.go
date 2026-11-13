package gasprice

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestADroppedConnectionIsTriedOnceMore(t *testing.T) {
	was := retryPause
	retryPause = time.Millisecond
	t.Cleanup(func() { retryPause = was })
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 { // drop the connection without an answer
			hj, _ := w.(http.Hijacker)
			c, _, _ := hj.Hijack()
			c.Close()
			return
		}
		w.Write([]byte(`{"response":{"data":[{"period":"2026-11-09","value":"3.10"}]}}`))
	}))
	defer srv.Close()
	pts, err := Fetch(context.Background(), nil, srv.URL, "k", ForState("MN"), time.Now())
	if err != nil || len(pts) != 1 || calls != 2 {
		t.Fatalf("%v %v after %d calls", pts, err, calls)
	}
}
