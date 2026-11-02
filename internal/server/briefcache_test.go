package server

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestASlowReadIsReusedBrieflyUntilAnythingIsWrittenAndErrorsAreNeverKept(t *testing.T) {
	briefDisabled = false // this test is about the reuse
	defer func() { briefDisabled = true }()
	s := &Server{}
	var calls int32
	h := s.briefly(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		w.Header().Set("X-Test", "yes")
		if r.URL.Query().Get("fail") != "" {
			http.Error(w, "nope", 500)
			return
		}
		w.Write([]byte{byte('0' + n)})
	})
	get := func(path string) (int, string, string) {
		rec := httptest.NewRecorder()
		h(rec, httptest.NewRequest("GET", path, nil))
		return rec.Code, rec.Body.String(), rec.Header().Get("X-Test")
	}
	if c, b, hd := get("/a"); c != 200 || b != "1" || hd != "yes" {
		t.Fatalf("%d %q %q", c, b, hd)
	}
	if _, b, hd := get("/a"); b != "1" || hd != "yes" || atomic.LoadInt32(&calls) != 1 {
		t.Fatalf("the second read is the kept answer: %q, %d calls", b, calls)
	}
	if _, b, _ := get("/b"); b != "2" {
		t.Fatalf("another address is its own answer: %q", b)
	}
	wrapped := s.writes(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	wrapped.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/api/anything", nil))
	if _, b, _ := get("/a"); b != "3" {
		t.Fatalf("a write throws the kept answers away: %q", b)
	}
	get("/c?fail=1")
	if c, _, _ := get("/c?fail=1"); c != 500 || atomic.LoadInt32(&calls) != 5 {
		t.Fatalf("an error is not kept, so it is asked for again: %d, %d calls", c, calls)
	}
}
