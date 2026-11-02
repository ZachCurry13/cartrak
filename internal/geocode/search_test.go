package geocode

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSearchReturnsMatchesAndSkipsBadOnes(t *testing.T) {
	var gotQ, gotAgent string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQ, gotAgent = r.URL.Query().Get("q"), r.Header.Get("User-Agent")
		w.Write([]byte(`[{"lat":"44.97","lon":"-93.26","display_name":"1 Main St, Townsville"},{"lat":"x","lon":"1","display_name":"bad"},{"lat":"95","lon":"1","display_name":"off the map"}]`))
	}))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, UserAgent: "test", MinGap: time.Millisecond}
	got, err := c.Search(context.Background(), "1 Main St Townsville")
	if err != nil || len(got) != 1 || got[0].Label != "1 Main St, Townsville" || got[0].Lat != 44.97 || got[0].Lon != -93.26 {
		t.Fatalf("got %+v, %v", got, err)
	}
	if gotQ != "1 Main St Townsville" || gotAgent != "test" {
		t.Fatalf("asked %q as %q", gotQ, gotAgent)
	}
}

func TestSearchRejectsAnEmptyOrHugeQueryWithoutAsking(t *testing.T) {
	c := &Client{BaseURL: "http://127.0.0.1:1", MinGap: time.Millisecond}
	for _, q := range []string{"", "  ", "ab", string(make([]byte, 300))} {
		if _, err := c.Search(context.Background(), q); err == nil {
			t.Fatalf("%q was accepted", q)
		}
	}
}

func TestSearchReportsAServiceError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(503) }))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, MinGap: time.Millisecond}
	if _, err := c.Search(context.Background(), "1 Main St"); err == nil {
		t.Fatal("a 503 was not an error")
	}
}
