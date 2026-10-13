package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestParseReturnForms(t *testing.T) {
	cases := []struct{ in, code, state string }{
		{"http://localhost/?code=A&state=S", "A", "S"},
		{"https://cartrak.example.com/ford/callback?state=S&code=A", "A", "S"},
		{"?code=A&state=S", "A", "S"},
		{"code=A", "A", ""},
		{"  \"http://localhost/?code=A\"  ", "A", ""},
		{"rawcode", "rawcode", ""},
	}
	for _, c := range cases {
		if got := parseReturn(c.in); got.Code != c.code || got.State != c.state {
			t.Errorf("parseReturn(%q) = %+v", c.in, got)
		}
	}
}

func TestParseReturnFragment(t *testing.T) {
	cases := []struct{ in, code, state, errCode string }{
		{"https://cartrak.example.com/ford/callback#code=A&state=S", "A", "S", ""},
		{"https://cartrak.example.com/ford/callback?x=1#code=A&state=S", "A", "S", ""},
		{"http://localhost/#code=A", "A", "", ""},
		{"#code=A&state=S", "A", "S", ""},
		{"http://localhost/?state=S#code=A", "A", "S", ""}, // split between query and fragment
		{"http://localhost/#error=access_denied&error_description=nope", "", "", "access_denied"},
		{"", "", "", ""},
	}
	for _, c := range cases {
		got := parseReturn(c.in)
		if got.Code != c.code || got.State != c.state || got.Err != c.errCode {
			t.Errorf("parseReturn(%q) = %+v", c.in, got)
		}
	}
}

func TestCallbackPageNeedsNoLoginAndLoadsItsScript(t *testing.T) {
	ts, _, _, _ := linkServer(t)
	resp, err := http.Get(ts.URL + "/ford/callback?code=x&state=y") // anonymous browser
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body := new(strings.Builder)
	buf := make([]byte, 2048)
	for {
		n, e := resp.Body.Read(buf)
		body.Write(buf[:n])
		if e != nil {
			break
		}
	}
	if resp.StatusCode != 200 || !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/html") ||
		!strings.Contains(body.String(), "/js/callback.js") || resp.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("callback page: %d %v %.80s", resp.StatusCode, resp.Header, body.String())
	}
	// The script it loads is served too.
	if js, err := http.Get(ts.URL + "/js/callback.js"); err != nil || js.StatusCode != 200 {
		t.Fatalf("callback.js not served: %v", err)
	}
}
