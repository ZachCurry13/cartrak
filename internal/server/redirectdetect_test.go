package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/poller"
	"github.com/ZachCurry13/cartrak/internal/settings"
	"github.com/ZachCurry13/cartrak/internal/store"
	"github.com/ZachCurry13/cartrak/internal/trips"
)

type fakeChecker struct {
	registered map[string]bool
	asked      []string
	err        error
}

func (f *fakeChecker) CheckRedirect(_ context.Context, uri string) (ford.RedirectCheck, error) {
	f.asked = append(f.asked, uri)
	if f.err != nil {
		return ford.RedirectCheck{}, f.err
	}
	return ford.RedirectCheck{URI: uri, Accepted: f.registered[uri]}, nil
}

func detectServer(t *testing.T, env map[string]string, fc *fakeChecker) (*httptest.Server, *settings.Settings) {
	s, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	cfg := settings.New(s, func(k string) (string, string) {
		if v := env[k]; v != "" {
			return v, "env"
		}
		return "", ""
	})
	cfg.Set("CARTRAK_PUBLIC_ADDRESS", "cartrak.example.com")
	d := &trips.Detector{S: s, TZFn: cfg.Location}
	srv := &Server{S: s, D: d, P: poller.New(fakeProv{}, s, d, poller.Intervals{}), Cfg: cfg, Redirects: fc}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts, cfg
}

func TestDetectPicksTheRegisteredAddress(t *testing.T) {
	// Registered on the Ford app: the bare site address and http://localhost (like the real case).
	fc := &fakeChecker{registered: map[string]bool{"https://cartrak.example.com": true, "http://localhost": true}}
	ts, cfg := detectServer(t, nil, fc)
	c := adminClient(t, ts.URL)
	cfg.Set("FORD_REDIRECT_URL", "https://cartrak.example.com/ford/callback") // the wrong value, as it was

	code, body := do(t, c, "POST", ts.URL+"/api/admin/ford/detect-redirect", "", xh)
	if code != 200 || !strings.Contains(body, `"chosen":"https://cartrak.example.com"`) || !strings.Contains(body, `"changed":true`) {
		t.Fatalf("detect: %d %s", code, body)
	}
	if cfg.Value("FORD_REDIRECT_URL") != "https://cartrak.example.com" {
		t.Fatalf("setting not switched: %q", cfg.Value("FORD_REDIRECT_URL"))
	}
	want := []string{"https://cartrak.example.com", "https://cartrak.example.com/ford/callback", "http://localhost"}
	if strings.Join(fc.asked, ",") != strings.Join(want, ",") {
		t.Fatalf("asked %v, want %v (never plain-http LAN addresses)", fc.asked, want)
	}
	// Running it again changes nothing.
	if _, body := do(t, c, "POST", ts.URL+"/api/admin/ford/detect-redirect", "", xh); !strings.Contains(body, `"changed":false`) {
		t.Fatalf("second run should not change anything: %s", body)
	}
}

func TestDetectKeepsAnAcceptedCurrentValue(t *testing.T) {
	fc := &fakeChecker{registered: map[string]bool{"https://cartrak.example.com": true, "http://localhost": true}}
	ts, cfg := detectServer(t, nil, fc)
	c := adminClient(t, ts.URL)
	cfg.Set("FORD_REDIRECT_URL", "http://localhost") // a deliberate choice that Ford accepts
	_, body := do(t, c, "POST", ts.URL+"/api/admin/ford/detect-redirect", "", xh)
	if !strings.Contains(body, `"changed":false`) || cfg.Value("FORD_REDIRECT_URL") != "http://localhost" {
		t.Fatalf("an accepted current value must be kept: %s", body)
	}
}

func TestDetectNoneAcceptedLockedAndErrors(t *testing.T) {
	// Nothing registered: report it, change nothing.
	fc := &fakeChecker{registered: map[string]bool{}}
	ts, cfg := detectServer(t, nil, fc)
	c := adminClient(t, ts.URL)
	cfg.Set("FORD_REDIRECT_URL", "https://cartrak.example.com/ford/callback")
	_, body := do(t, c, "POST", ts.URL+"/api/admin/ford/detect-redirect", "", xh)
	if !strings.Contains(body, `"chosen":""`) || !strings.Contains(body, `"changed":false`) || cfg.Value("FORD_REDIRECT_URL") != "https://cartrak.example.com/ford/callback" {
		t.Fatalf("none accepted: %s", body)
	}

	// Locked by an environment variable: report, never overwrite.
	fc2 := &fakeChecker{registered: map[string]bool{"https://cartrak.example.com": true}}
	ts2, cfg2 := detectServer(t, map[string]string{"FORD_REDIRECT_URL": "https://cartrak.example.com/ford/callback"}, fc2)
	c2 := adminClient(t, ts2.URL)
	_, body2 := do(t, c2, "POST", ts2.URL+"/api/admin/ford/detect-redirect", "", xh)
	if !strings.Contains(body2, `"locked":true`) || !strings.Contains(body2, `"changed":false`) || cfg2.Value("FORD_REDIRECT_URL") != "https://cartrak.example.com/ford/callback" {
		t.Fatalf("locked value overwritten: %s", body2)
	}

	// Ford unreachable: an error, not a false "not registered".
	fc3 := &fakeChecker{err: errors.New("dial tcp: no route")}
	ts3, _ := detectServer(t, nil, fc3)
	c3 := adminClient(t, ts3.URL)
	if code, _ := do(t, c3, "POST", ts3.URL+"/api/admin/ford/detect-redirect", "", xh); code != 502 {
		t.Fatalf("unreachable Ford should be 502, got %d", code)
	}
}

func TestDetectAdminOnly(t *testing.T) {
	ts, _ := detectServer(t, nil, &fakeChecker{registered: map[string]bool{}})
	c := adminClient(t, ts.URL)
	do(t, c, "POST", ts.URL+"/api/users", `{"name":"fam","password":"viewer-password-1","role":"viewer"}`, xh)
	vj, _ := cookieJar()
	v := &http.Client{Jar: vj}
	do(t, v, "POST", ts.URL+"/api/login", `{"name":"fam","password":"viewer-password-1"}`, nil)
	if code, _ := do(t, v, "POST", ts.URL+"/api/admin/ford/detect-redirect", "", xh); code != 403 {
		t.Fatalf("viewer ran the check: %d", code)
	}
}

// reflectLinker reports the redirect address it would send to Ford.
type reflectLinker struct{ cfg *settings.Settings }

func (l reflectLinker) LinkURL(state string) string {
	return "https://ford.example/authorize?redirect_uri=" + l.cfg.Value("FORD_REDIRECT_URL") + "&state=" + state
}
func (reflectLinker) Exchange(context.Context, string) error { return nil }

func connectServer(t *testing.T, fc *fakeChecker) (*httptest.Server, *settings.Settings) {
	s, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	cfg := settings.New(s, func(string) (string, string) { return "", "" })
	cfg.Set("CARTRAK_PUBLIC_ADDRESS", "cartrak.example.com")
	cfg.Set("FORD_CLIENT_ID", "id")
	cfg.Set("FORD_CLIENT_SECRET", "s")
	d := &trips.Detector{S: s, TZFn: cfg.Location}
	srv := &Server{S: s, D: d, P: poller.New(fakeProv{}, s, d, poller.Intervals{}), Cfg: cfg, Redirects: fc, Link: reflectLinker{cfg}}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts, cfg
}

// Pressing Connect Ford must never send Ford a redirect address it is known to reject.
func TestConnectSwitchesToAnAcceptedAddressFirst(t *testing.T) {
	fc := &fakeChecker{registered: map[string]bool{"https://cartrak.example.com": true}}
	ts, cfg := connectServer(t, fc)
	cfg.Set("FORD_REDIRECT_URL", "https://cartrak.example.com/ford/callback") // the wrong value, as it was
	c := adminClient(t, ts.URL)

	resp, err := c.Get(ts.URL + "/ford/connect")
	if err != nil || resp.StatusCode != 302 {
		t.Fatalf("connect: %v %v", err, resp)
	}
	loc := resp.Header.Get("Location")
	if !strings.HasPrefix(loc, "https://ford.example/authorize?redirect_uri=https://cartrak.example.com&state=") {
		t.Fatalf("Ford was sent the wrong redirect: %s", loc)
	}
	if cfg.Value("FORD_REDIRECT_URL") != "https://cartrak.example.com" {
		t.Fatalf("setting not switched: %q", cfg.Value("FORD_REDIRECT_URL"))
	}
}

func TestConnectLeavesAnAcceptedAddressAlone(t *testing.T) {
	fc := &fakeChecker{registered: map[string]bool{"https://cartrak.example.com": true}}
	ts, cfg := connectServer(t, fc)
	cfg.Set("FORD_REDIRECT_URL", "https://cartrak.example.com")
	c := adminClient(t, ts.URL)
	resp, _ := c.Get(ts.URL + "/ford/connect")
	if len(fc.asked) != 1 || !strings.Contains(resp.Header.Get("Location"), "redirect_uri=https://cartrak.example.com&") {
		t.Fatalf("an accepted address needs just one check: asked=%v loc=%s", fc.asked, resp.Header.Get("Location"))
	}
}

func TestConnectWithNothingAcceptedExplainsInsteadOfSendingToFord(t *testing.T) {
	fc := &fakeChecker{registered: map[string]bool{}}
	ts, cfg := connectServer(t, fc)
	cfg.Set("FORD_REDIRECT_URL", "https://cartrak.example.com/ford/callback")
	c := adminClient(t, ts.URL)
	resp, _ := c.Get(ts.URL + "/ford/connect")
	if resp.StatusCode != 302 || resp.Header.Get("Location") != "/#/connect" {
		t.Fatalf("should go to the Connect page, not Ford: %d %s", resp.StatusCode, resp.Header.Get("Location"))
	}
}

func TestConnectProceedsWhenFordCannotBeReached(t *testing.T) {
	fc := &fakeChecker{err: errors.New("offline")}
	ts, cfg := connectServer(t, fc)
	cfg.Set("FORD_REDIRECT_URL", "https://cartrak.example.com")
	c := adminClient(t, ts.URL)
	resp, _ := c.Get(ts.URL + "/ford/connect")
	if !strings.HasPrefix(resp.Header.Get("Location"), "https://ford.example/authorize") {
		t.Fatalf("an unreachable checker must not block sign-in: %s", resp.Header.Get("Location"))
	}
}
