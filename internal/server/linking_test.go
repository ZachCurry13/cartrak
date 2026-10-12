package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/poller"
	"github.com/ZachCurry13/cartrak/internal/store"
	"github.com/ZachCurry13/cartrak/internal/trips"
)

type recLinker struct {
	code    string
	calls   int
	failErr error
}

func (*recLinker) LinkURL(state string) string {
	return "https://ford.example/authorize?state=" + state
}
func (l *recLinker) Exchange(_ context.Context, code string) error {
	l.calls++
	l.code = code
	return l.failErr
}

func linkServer(t *testing.T) (*httptest.Server, *Server, *recLinker, *http.Client) {
	s, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	d := &trips.Detector{S: s, TZ: time.UTC}
	rl := &recLinker{}
	srv := &Server{S: s, D: d, P: poller.New(fakeProv{}, s, d, poller.Intervals{}), Link: rl}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts, srv, rl, adminClient(t, ts.URL)
}

// startLink presses "Connect Ford" and returns the state value Ford would echo back.
func startLink(t *testing.T, c *http.Client, base string) string {
	resp, err := c.Get(base + "/ford/connect")
	if err != nil || resp.StatusCode != 302 {
		t.Fatalf("connect: %v %v", err, resp)
	}
	u, _ := url.Parse(resp.Header.Get("Location"))
	if u.Query().Get("state") == "" {
		t.Fatalf("no state in %s", u)
	}
	return u.Query().Get("state")
}

func complete(t *testing.T, c *http.Client, base, pasted string) (int, string) {
	b, _ := jsonString(pasted)
	return do(t, c, "POST", base+"/api/ford/complete", `{"url":`+b+`}`, xh)
}

func jsonString(s string) (string, error) {
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`, "\r", `\r`)
	return `"` + r.Replace(s) + `"`, nil
}

func TestPasteFullAddressLinks(t *testing.T) {
	ts, _, rl, c := linkServer(t)
	state := startLink(t, c, ts.URL)
	code, body := complete(t, c, ts.URL, "http://localhost/?code=the-code-123&state="+state)
	if code != 200 || rl.code != "the-code-123" {
		t.Fatalf("paste failed: %d %s (code=%q)", code, body, rl.code)
	}
	// The state is single-use.
	if code, _ := complete(t, c, ts.URL, "http://localhost/?code=again&state="+state); code != 400 || rl.calls != 1 {
		t.Fatalf("state reused: %d calls=%d", code, rl.calls)
	}
}

func TestPasteQueryOnlyAndBareCode(t *testing.T) {
	ts, _, rl, c := linkServer(t)
	state := startLink(t, c, ts.URL)
	if code, _ := complete(t, c, ts.URL, "?code=q-code&state="+state); code != 200 || rl.code != "q-code" {
		t.Fatalf("query-only: %d %q", code, rl.code)
	}
	startLink(t, c, ts.URL)
	if code, _ := complete(t, c, ts.URL, "  just-a-code  "); code != 200 || rl.code != "just-a-code" {
		t.Fatalf("bare code: %d %q", code, rl.code)
	}
}

func TestPasteRejectsWrongStateAndNoPendingLink(t *testing.T) {
	ts, _, rl, c := linkServer(t)
	if code, _ := complete(t, c, ts.URL, "http://localhost/?code=x&state=whatever"); code != 400 || rl.calls != 0 {
		t.Fatalf("no pending link should be refused: %d", code)
	}
	startLink(t, c, ts.URL)
	if code, body := complete(t, c, ts.URL, "http://localhost/?code=x&state=forged"); code != 400 || rl.calls != 0 || !strings.Contains(body, "different sign-in") {
		t.Fatalf("forged state accepted: %d %s", code, body)
	}
}

func TestPasteExpiredLink(t *testing.T) {
	ts, srv, rl, c := linkServer(t)
	state := startLink(t, c, ts.URL)
	srv.S.SetKV("ford_state", state+"|1") // expired long ago
	if code, _ := complete(t, c, ts.URL, "http://localhost/?code=x&state="+state); code != 400 || rl.calls != 0 {
		t.Fatalf("expired link accepted: %d", code)
	}
}

func TestPasteTheErrorFromFord(t *testing.T) {
	ts, _, rl, c := linkServer(t)
	startLink(t, c, ts.URL)
	// The address a mis-registered redirect produced (host and ids changed).
	pasted := "http://localhost/?error=redirect_uri_mismatch&error_description=AADB2C90006%3a+The+redirect+URI+%27http%3a%2f%2fnas.example%3a1%2fford%2fcallback%27+provided+in+the+request+is+not+registered+for+the+client+id+%27abc%27.%0d%0aCorrelation+ID%3a+251526e1%0d%0aTimestamp%3a+2026-10-12&state=zzz"
	code, body := complete(t, c, ts.URL, pasted)
	if code != 400 || rl.calls != 0 {
		t.Fatalf("error address should be refused: %d", code)
	}
	for _, want := range []string{"redirect_uri_mismatch", "AADB2C90006", "must exactly match", "http://localhost"} {
		if !strings.Contains(body, want) {
			t.Errorf("message missing %q: %s", want, body)
		}
	}
	if strings.Contains(body, "Correlation") || strings.Contains(body, "251526e1") {
		t.Errorf("correlation id should be dropped: %s", body)
	}
}

func TestExchangeFailureIsReportedWithoutSecrets(t *testing.T) {
	ts, _, rl, c := linkServer(t)
	rl.failErr = errors.New("ford token: HTTP 400 (invalid_grant)")
	state := startLink(t, c, ts.URL)
	code, body := complete(t, c, ts.URL, "http://localhost/?code=c&state="+state)
	if code != 502 || !strings.Contains(body, "invalid_grant") {
		t.Fatalf("exchange error not shown: %d %s", code, body)
	}
}

func TestPasteIsAdminOnly(t *testing.T) {
	ts, _, rl, c := linkServer(t)
	startLink(t, c, ts.URL)
	do(t, c, "POST", ts.URL+"/api/users", `{"name":"fam","password":"viewer-password-1","role":"viewer"}`, xh)
	vj, _ := cookieJar()
	v := &http.Client{Jar: vj}
	do(t, v, "POST", ts.URL+"/api/login", `{"name":"fam","password":"viewer-password-1"}`, nil)
	if code, _ := complete(t, v, ts.URL, "http://localhost/?code=x"); code != 403 || rl.calls != 0 {
		t.Fatalf("viewer linked the car: %d", code)
	}
	if code, _ := do(t, c, "POST", ts.URL+"/api/ford/complete", `{"url":"x"}`, nil); code != 403 {
		t.Fatalf("missing CSRF header should be refused: %d", code)
	}
}

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
