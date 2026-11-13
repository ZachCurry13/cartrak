package server

import (
	"net/http"
	"strings"
	"testing"
)

// Every answer carries the browser protections: the page, the health check
// and the API, signed in or not.
func TestSecurityHeaders(t *testing.T) {
	ts, _ := newTestServer(t)
	want := map[string]string{
		"X-Content-Type-Options":    "nosniff",
		"X-Frame-Options":           "DENY",
		"Referrer-Policy":           "same-origin",
		"Strict-Transport-Security": "max-age=63072000; includeSubDomains",
	}
	for _, path := range []string{"/", "/healthz", "/api/me"} {
		resp, err := http.Get(ts.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		for name, value := range want {
			if got := resp.Header.Get(name); got != value {
				t.Errorf("%s: %s = %q, want %q", path, name, got, value)
			}
		}
		policy := resp.Header.Get("Content-Security-Policy")
		for _, part := range []string{"default-src 'self'", "frame-ancestors 'none'", "base-uri 'self'"} {
			if !strings.Contains(policy, part) {
				t.Errorf("%s: the content policy %q lacks %q", path, policy, part)
			}
		}
		if strings.HasPrefix(path, "/api/") && resp.Header.Get("Cache-Control") != "no-store" {
			t.Errorf("%s: API answers must not be kept, got Cache-Control %q", path, resp.Header.Get("Cache-Control"))
		}
	}
}
