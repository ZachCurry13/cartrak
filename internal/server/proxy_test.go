package server

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientIPTrustProxy(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "10.0.0.5:1234"
	r.Header.Set("CF-Connecting-IP", "203.0.113.9")

	if got := (&Server{TrustProxy: true}).clientIP(r); got != "203.0.113.9" {
		t.Fatalf("trusted proxy header ignored: %s", got)
	}
	// Untrusted: a spoofed header must not let a caller dodge the lockout.
	if got := (&Server{}).clientIP(r); got != "10.0.0.5" {
		t.Fatalf("spoofable header believed: %s", got)
	}
	r.Header.Del("CF-Connecting-IP")
	r.Header.Set("X-Forwarded-For", "198.51.100.7, 10.0.0.5")
	if got := (&Server{TrustProxy: true}).clientIP(r); got != "198.51.100.7" {
		t.Fatalf("x-forwarded-for first hop: %s", got)
	}
}

func TestSessionTTL(t *testing.T) {
	if (&Server{}).sessionTTL() != 30*24*time.Hour || (&Server{SessionDays: 7}).sessionTTL() != 7*24*time.Hour {
		t.Fatal("bad session ttl")
	}
}
