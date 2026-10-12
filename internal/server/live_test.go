package server

import (
	"bufio"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/poller"
	"github.com/ZachCurry13/cartrak/internal/store"
	"github.com/ZachCurry13/cartrak/internal/trips"
)

func liveServer(t *testing.T) (*httptest.Server, *Server, *store.Store) {
	s, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	d := &trips.Detector{S: s, TZ: time.UTC}
	srv := &Server{S: s, D: d, P: poller.New(fakeProv{}, s, d, poller.Intervals{})}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts, srv, s
}

func saveReading(t *testing.T, s *store.Store, soc float64, ts time.Time) {
	odo := 100.0
	st := &ford.State{TS: ts, SoCPct: &soc, OdometerKm: &odo, Ignition: "OFF", Raw: []byte(`{}`),
		Extra: map[string]any{"some.long.field": "x"}}
	if _, err := s.SaveState("test", st); err != nil {
		t.Fatal(err)
	}
}

func TestGzipAndETag(t *testing.T) {
	ts, _, s := liveServer(t)
	c := adminClient(t, ts.URL)
	saveReading(t, s, 77, time.Now())

	req, _ := http.NewRequest("GET", ts.URL+"/api/latest", nil)
	req.Header.Set("Accept-Encoding", "gzip") // setting it ourselves turns off Go's automatic decoding
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Header.Get("Content-Encoding") != "gzip" {
		t.Fatalf("JSON not compressed: %v", resp.Header)
	}
	zr, err := gzip.NewReader(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(zr)
	resp.Body.Close()
	etag := resp.Header.Get("ETag")
	if !strings.Contains(string(body), `"socPct":77`) || etag == "" {
		t.Fatalf("bad body/etag: %s %q", body, etag)
	}

	// Unchanged data: a 304 with no body.
	req2, _ := http.NewRequest("GET", ts.URL+"/api/latest", nil)
	req2.Header.Set("If-None-Match", etag)
	resp2, _ := c.Do(req2)
	if resp2.StatusCode != 304 {
		t.Fatalf("want 304, got %d", resp2.StatusCode)
	}
	resp2.Body.Close()

	// The summary is smaller and omits the long field list.
	_, full := do(t, c, "GET", ts.URL+"/api/latest", "", nil)
	_, sum := do(t, c, "GET", ts.URL+"/api/latest?summary=1", "", nil)
	if !strings.Contains(full, "some.long.field") || strings.Contains(sum, "some.long.field") || len(sum) >= len(full) {
		t.Fatalf("summary should be smaller:\nfull=%s\nsum=%s", full, sum)
	}

	// Plain clients still get plain text.
	plain, _ := http.NewRequest("GET", ts.URL+"/", nil)
	plain.Header.Set("Accept-Encoding", "identity")
	r3, _ := c.Do(plain)
	if r3.Header.Get("Content-Encoding") != "" {
		t.Fatal("compressed without being asked")
	}
	r3.Body.Close()
}

func TestLiveStreamPushesNewReadings(t *testing.T) {
	ts, srv, s := liveServer(t)
	c := adminClient(t, ts.URL)
	saveReading(t, s, 50, time.Now())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", ts.URL+"/api/live", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.Header.Get("Content-Type") != "text/event-stream" || resp.Header.Get("Content-Encoding") != "" {
		t.Fatalf("stream headers wrong: %v", resp.Header)
	}
	rd := bufio.NewReader(resp.Body)
	next := func() string { // one SSE message
		var data string
		for {
			line, err := rd.ReadString('\n')
			if err != nil {
				t.Fatalf("stream ended: %v", err)
			}
			if strings.HasPrefix(line, "data: ") {
				data = line
			}
			if line == "\n" && data != "" {
				return data
			}
		}
	}
	if first := next(); !strings.Contains(first, `"socPct":50`) {
		t.Fatalf("first event should be the current reading: %s", first)
	}
	saveReading(t, s, 49, time.Now().Add(time.Minute))
	for i := 0; i < 50 && srv.hub.count() == 0; i++ {
		time.Sleep(20 * time.Millisecond)
	}
	srv.Broadcast()
	if second := next(); !strings.Contains(second, `"socPct":49`) {
		t.Fatalf("pushed event wrong: %s", second)
	}
}

func TestLiveModeAdminOnly(t *testing.T) {
	ts, srv, _ := liveServer(t)
	c := adminClient(t, ts.URL)
	if code, _ := do(t, c, "POST", ts.URL+"/api/live-mode", `{"enabled":true}`, xh); code != 200 {
		t.Fatalf("admin live-mode = %d", code)
	}
	if !srv.P.Status().Live {
		t.Fatal("live mode not on")
	}
	do(t, c, "POST", ts.URL+"/api/users", `{"name":"fam","password":"viewer-password-1","role":"viewer"}`, xh)
	vj, _ := cookieJar()
	v := &http.Client{Jar: vj}
	do(t, v, "POST", ts.URL+"/api/login", `{"name":"fam","password":"viewer-password-1"}`, nil)
	if code, _ := do(t, v, "POST", ts.URL+"/api/live-mode", `{"enabled":false}`, xh); code != 403 {
		t.Fatalf("viewer toggled live mode = %d", code)
	}
	do(t, c, "POST", ts.URL+"/api/live-mode", `{"enabled":false}`, xh)
	if srv.P.Status().Live {
		t.Fatal("live mode not off")
	}
}
