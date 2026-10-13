package ford

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestNotLinkedV2(t *testing.T) {
	f := newFake2(t)
	c := f.client(memKV{})
	if c.Ready() {
		t.Fatal("ready without a token")
	}
	if _, err := c.Fetch(context.Background()); !errors.Is(err, ErrNotLinked) {
		t.Fatalf("want ErrNotLinked, got %v", err)
	}
}

func TestTokenErrorDoesNotLeakBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		w.Write([]byte(`{"error":"invalid_grant","error_description":"AADB2C: the secret sec-123 is wrong"}`))
	}))
	defer srv.Close()
	c := NewConnect2(func() Connect2Opts {
		return Connect2Opts{ClientID: "cid", ClientSecret: "sec-123", TokenURL: srv.URL}
	}, memKV{})
	err := c.Exchange(context.Background(), "x")
	if err == nil || strings.Contains(err.Error(), "sec-123") || !strings.Contains(err.Error(), "invalid_grant") {
		t.Fatalf("error should show only the short code: %v", err)
	}
}

// Some Ford apps reject the scope field on the token request; the exchange then retries without it.
func TestTokenExchangeRetriesWithoutScopeOn400(t *testing.T) {
	var forms []url.Values
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		forms = append(forms, r.PostForm)
		if r.PostForm.Get("scope") != "" {
			w.WriteHeader(400)
			w.Write([]byte(`{"error":"invalid_scope"}`))
			return
		}
		w.Write([]byte(`{"access_token":"A","refresh_token":"R","expires_in":1200}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	kv := memKV{}
	c := NewConnect2(func() Connect2Opts {
		return Connect2Opts{ClientID: "cid", ClientSecret: "sec", RedirectURL: "https://x.example", TokenURL: srv.URL + "/token"}
	}, kv)
	if err := c.Exchange(context.Background(), "code1"); err != nil || kv[refreshKey] != "R" {
		t.Fatalf("exchange should succeed on the retry: %v %v", err, kv)
	}
	if len(forms) != 2 || forms[0].Get("scope") == "" || forms[1].Get("scope") != "" || forms[1].Get("code") != "code1" {
		t.Fatalf("expected a with-scope try then a no-scope retry: %v", forms)
	}
}

func TestTokenErrorsOtherThan400AreNotRetried(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits++; w.WriteHeader(401) }))
	defer srv.Close()
	c := NewConnect2(func() Connect2Opts { return Connect2Opts{ClientID: "c", ClientSecret: "s", TokenURL: srv.URL} }, memKV{})
	if err := c.Exchange(context.Background(), "x"); err == nil || hits != 1 {
		t.Fatalf("a 401 must fail once, not retry: err=%v hits=%d", err, hits)
	}
}

// The real telemetry shape (as read by other FordConnect Query clients):
