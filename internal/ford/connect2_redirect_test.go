package ford

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeAuth answers like Ford's sign-in service: a registered redirect proceeds
// (here to a server_error, as it does for a request with no signed-in user), any
// other gets redirect_uri_mismatch.
func fakeAuth(t *testing.T, registered ...string) *Connect2 {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ru := r.URL.Query().Get("redirect_uri")
		for _, ok := range registered {
			if ru == ok {
				w.Header().Set("Location", ru+"?error=server_error&error_description=AADB2C90075%3a+claims+exchange+failed%0d%0aCorrelation+ID%3a+x")
				w.WriteHeader(302)
				return
			}
		}
		w.Header().Set("Location", "http://localhost/?error=redirect_uri_mismatch&error_description=AADB2C90006%3a+The+redirect+URI+is+not+registered")
		w.WriteHeader(302)
	}))
	t.Cleanup(srv.Close)
	return NewConnect2(func() Connect2Opts { return Connect2Opts{ClientID: "cid", AuthURL: srv.URL + "/authorize?p=P"} }, memKV{})
}

func TestCheckRedirectAcceptedAndRejected(t *testing.T) {
	c := fakeAuth(t, "https://cartrak.example.com", "http://localhost")
	cases := map[string]bool{
		"https://cartrak.example.com":               true,
		"http://localhost":                          true,
		"https://cartrak.example.com/ford/callback": false,
		"https://other.example.com":                 false,
	}
	for uri, want := range cases {
		got, err := c.CheckRedirect(context.Background(), uri)
		if err != nil || got.Accepted != want {
			t.Errorf("%s: accepted=%v err=%v detail=%q, want %v", uri, got.Accepted, err, got.Detail, want)
		}
	}
	acc, _ := c.CheckRedirect(context.Background(), "https://cartrak.example.com")
	if !strings.Contains(acc.Detail, "AADB2C90075") || strings.Contains(acc.Detail, "Correlation") {
		t.Errorf("accepted detail should mention Ford's other problem, without the correlation id: %q", acc.Detail)
	}
	rej, _ := c.CheckRedirect(context.Background(), "https://nope.example.com")
	if !strings.Contains(rej.Detail, "not registered") {
		t.Errorf("rejected detail: %q", rej.Detail)
	}
}

func TestCheckRedirectSendsNoSecretAndNeverFollowsRedirects(t *testing.T) {
	var seen string
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		seen = r.URL.RawQuery + r.Header.Get("Authorization")
		w.Header().Set("Location", "http://localhost/?error=redirect_uri_mismatch")
		w.WriteHeader(302)
	}))
	defer srv.Close()
	c := NewConnect2(func() Connect2Opts {
		return Connect2Opts{ClientID: "cid", ClientSecret: "TOP-SECRET", AuthURL: srv.URL + "/a"}
	}, memKV{})
	if _, err := c.CheckRedirect(context.Background(), "https://x.example.com"); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(seen, "TOP-SECRET") || !strings.Contains(seen, "client_id=cid") || hits != 1 {
		t.Fatalf("request should carry the client id but never the secret, one hit only: hits=%d %s", hits, seen)
	}
}

func TestCheckRedirectNetworkErrorIsAnError(t *testing.T) {
	c := NewConnect2(func() Connect2Opts { return Connect2Opts{ClientID: "c", AuthURL: "http://127.0.0.1:1/a"} }, memKV{})
	if _, err := c.CheckRedirect(context.Background(), "https://x.example.com"); err == nil {
		t.Fatal("an unreachable Ford must be an error, not 'not registered'")
	}
}

func TestSignInAddressTemplateFromFordDocs(t *testing.T) {
	c := NewConnect2(func() Connect2Opts {
		return Connect2Opts{ClientID: "cid", RedirectURL: "https://cartrak.example.com",
			AuthURL: "https://ford.example/link?make=F&client_id={client_id}&redirect_uri={redirect_uri}&state={state}&scope={scope}&extra=1"}
	}, memKV{})
	got := c.LinkURL("st8")
	want := "https://ford.example/link?make=F&client_id=cid&redirect_uri=https%3A%2F%2Fcartrak.example.com&state=st8&scope=cid+offline_access+openid&extra=1"
	if got != want {
		t.Fatalf("template not filled in:\n got %s\nwant %s", got, want)
	}
	// Without placeholders the standard parameters are appended as before.
	plain := NewConnect2(func() Connect2Opts {
		return Connect2Opts{ClientID: "cid", RedirectURL: "https://x.example", AuthURL: "https://ford.example/authorize?p=P"}
	}, memKV{})
	if u := plain.LinkURL("s"); !strings.HasPrefix(u, "https://ford.example/authorize?p=P&client_id=cid") || !strings.Contains(u, "redirect_uri=https%3A%2F%2Fx.example") {
		t.Fatalf("plain address: %s", u)
	}
}
