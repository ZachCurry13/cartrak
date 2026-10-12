package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/poller"
	"github.com/ZachCurry13/cartrak/internal/settings"
	"github.com/ZachCurry13/cartrak/internal/store"
	"github.com/ZachCurry13/cartrak/internal/trips"
)

type fakeLinker struct{}

func (fakeLinker) LinkURL(state string) string                     { return "https://ford.example/link?state=" + state }
func (fakeLinker) Exchange(ctx context.Context, code string) error { return nil }

func newCfgServer(t *testing.T, env map[string]string) (*httptest.Server, *settings.Settings) {
	s, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	proc := func(k string) (string, string) {
		if v := env[k]; v != "" {
			return v, "env"
		}
		return "", ""
	}
	cfg := settings.New(s, proc)
	d := &trips.Detector{S: s, TZFn: cfg.Location}
	srv := &Server{S: s, D: d, P: poller.New(fakeProv{}, s, d, poller.Intervals{}), Cfg: cfg, Link: fakeLinker{}}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts, cfg
}

func adminClient(t *testing.T, base string) *http.Client {
	jar, _ := cookieJar()
	c := &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	if code, _ := do(t, c, "POST", base+"/api/setup", `{"name":"owner","password":"correct-horse-battery"}`, nil); code != 200 {
		t.Fatalf("setup = %d", code)
	}
	return c
}

var xh = map[string]string{"X-Requested-With": "cartrak"}

func TestAdminSettingsSecretNeverReturned(t *testing.T) {
	ts, cfg := newCfgServer(t, nil)
	c := adminClient(t, ts.URL)

	body := `{"values":{"FORD_CLIENT_ID":"id-123","FORD_CLIENT_SECRET":"super-secret-value","FORD_APPLICATION_ID":"app-1","CARTRAK_TZ":"America/Chicago"}}`
	code, resp := do(t, c, "PUT", ts.URL+"/api/admin/settings", body, xh)
	if code != 200 || strings.Contains(resp, "super-secret-value") {
		t.Fatalf("save: %d, secret leaked=%v", code, strings.Contains(resp, "super-secret-value"))
	}
	if _, resp := do(t, c, "GET", ts.URL+"/api/admin/settings", "", nil); strings.Contains(resp, "super-secret-value") || !strings.Contains(resp, `"isSet":true`) {
		t.Fatalf("get leaked or lost the secret: %s", resp)
	}
	if cfg.Value("FORD_CLIENT_SECRET") != "super-secret-value" || !cfg.Configured() || cfg.Location().String() != "America/Chicago" {
		t.Fatal("values not stored")
	}
	// A blank secret means keep the current one.
	do(t, c, "PUT", ts.URL+"/api/admin/settings", `{"values":{"FORD_CLIENT_SECRET":"","FORD_CLIENT_ID":"id-456"}}`, xh)
	if cfg.Value("FORD_CLIENT_SECRET") != "super-secret-value" || cfg.Value("FORD_CLIENT_ID") != "id-456" {
		t.Fatal("blank secret should be kept, id updated")
	}
	do(t, c, "PUT", ts.URL+"/api/admin/settings", `{"clear":["FORD_CLIENT_SECRET"]}`, xh)
	if cfg.Value("FORD_CLIENT_SECRET") != "" {
		t.Fatal("clear failed")
	}
}

func TestAdminSettingsValidationLockingAndRoles(t *testing.T) {
	ts, _ := newCfgServer(t, map[string]string{"FORD_APPLICATION_ID": "env-app"})
	c := adminClient(t, ts.URL)

	if code, _ := do(t, c, "PUT", ts.URL+"/api/admin/settings", `{"values":{"CARTRAK_TZ":"Nope/Zone"}}`, xh); code != 400 {
		t.Fatalf("bad tz = %d", code)
	}
	if code, _ := do(t, c, "PUT", ts.URL+"/api/admin/settings", `{"values":{"FORD_REDIRECT_URL":"http://example.com/x"}}`, xh); code != 400 {
		t.Fatalf("insecure redirect = %d", code)
	}
	if code, resp := do(t, c, "PUT", ts.URL+"/api/admin/settings", `{"values":{"FORD_APPLICATION_ID":"x"}}`, xh); code != 409 || !strings.Contains(resp, "config.yaml") {
		t.Fatalf("locked key accepted: %d %s", code, resp)
	}
	if code, _ := do(t, c, "PUT", ts.URL+"/api/admin/settings", `{"values":{"NOT_A_KEY":"x"}}`, xh); code != 400 {
		t.Fatalf("unknown key = %d", code)
	}
	if _, resp := do(t, c, "GET", ts.URL+"/api/admin/settings", "", nil); !strings.Contains(resp, `"locked":true`) {
		t.Fatalf("locked flag missing: %s", resp)
	}

	do(t, c, "POST", ts.URL+"/api/users", `{"name":"fam","password":"viewer-password-1","role":"viewer"}`, xh)
	vj, _ := cookieJar()
	v := &http.Client{Jar: vj}
	do(t, v, "POST", ts.URL+"/api/login", `{"name":"fam","password":"viewer-password-1"}`, nil)
	if code, _ := do(t, v, "GET", ts.URL+"/api/admin/settings", "", nil); code != 403 {
		t.Fatalf("viewer read settings = %d", code)
	}
	if code, _ := do(t, v, "PUT", ts.URL+"/api/admin/settings", `{"values":{"FORD_VIN":"x"}}`, xh); code != 403 {
		t.Fatalf("viewer wrote settings = %d", code)
	}
}

func TestConnectNeedsCredentialsThenDerivesRedirect(t *testing.T) {
	ts, cfg := newCfgServer(t, nil)
	c := adminClient(t, ts.URL)

	req, _ := http.NewRequest("GET", ts.URL+"/ford/connect", nil)
	resp, err := c.Do(req)
	if err != nil || resp.StatusCode != 302 || resp.Header.Get("Location") != "/#/admin" {
		t.Fatalf("unconfigured connect should go to the admin page: %v %v", resp.StatusCode, resp.Header.Get("Location"))
	}

	do(t, c, "PUT", ts.URL+"/api/admin/settings", `{"values":{"FORD_CLIENT_ID":"a","FORD_CLIENT_SECRET":"b","FORD_APPLICATION_ID":"c"}}`, xh)
	resp, _ = c.Do(req)
	if resp.StatusCode != 302 || !strings.HasPrefix(resp.Header.Get("Location"), "https://ford.example/link") {
		t.Fatalf("configured connect should go to Ford: %v %v", resp.StatusCode, resp.Header.Get("Location"))
	}
	if got := cfg.Value("FORD_REDIRECT_URL"); !strings.HasSuffix(got, "/ford/callback") {
		t.Fatalf("redirect not derived: %q", got)
	}
	_ = time.Second
}

type readyProv struct{ fakeProv }

func (readyProv) Ready() bool { return true }

type fakeProber struct{}

func (fakeProber) Probe(context.Context) map[string]ford.ProbeResult {
	return map[string]ford.ProbeResult{"telemetry": {Status: 200, Bytes: 10, Body: map[string]any{"ok": true}}}
}

func TestFordProbeEndpoint(t *testing.T) {
	// Not linked yet: 409 for admins, 403 for viewers.
	ts, _ := newCfgServer(t, nil)
	ts2 := ts
	c := adminClient(t, ts2.URL)
	if code, _ := do(t, c, "POST", ts2.URL+"/api/admin/ford-probe", "", xh); code != 404 {
		t.Fatalf("no prober should be 404, got %d", code)
	}

	s, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	cfg := settings.New(s, func(string) (string, string) { return "", "" })
	d := &trips.Detector{S: s, TZFn: cfg.Location}
	linked := &Server{S: s, D: d, P: poller.New(readyProv{}, s, d, poller.Intervals{}), Cfg: cfg, Prober: fakeProber{}}
	srv := httptest.NewServer(linked.Handler())
	t.Cleanup(srv.Close)
	ac := adminClient(t, srv.URL)
	code, body := do(t, ac, "POST", srv.URL+"/api/admin/ford-probe", "", xh)
	if code != 200 || !strings.Contains(body, `"telemetry"`) || !strings.Contains(body, "redacted") {
		t.Fatalf("probe: %d %s", code, body)
	}
	if code, _ := do(t, ac, "POST", srv.URL+"/api/admin/ford-probe", "", nil); code != 403 {
		t.Fatalf("missing CSRF header should be refused, got %d", code)
	}
	do(t, ac, "POST", srv.URL+"/api/users", `{"name":"fam","password":"viewer-password-1","role":"viewer"}`, xh)
	vj, _ := cookieJar()
	v := &http.Client{Jar: vj}
	do(t, v, "POST", srv.URL+"/api/login", `{"name":"fam","password":"viewer-password-1"}`, nil)
	if code, _ := do(t, v, "POST", srv.URL+"/api/admin/ford-probe", "", xh); code != 403 {
		t.Fatalf("viewer probe = %d", code)
	}

	// Prober present but account not linked: 409.
	unlinked := &Server{S: s, D: d, P: poller.New(fakeProv{}, s, d, poller.Intervals{}), Cfg: cfg, Prober: fakeProber{}}
	srv3 := httptest.NewServer(unlinked.Handler())
	t.Cleanup(srv3.Close)
	if code, _ := do(t, ac, "POST", srv3.URL+"/api/admin/ford-probe", "", xh); code != 409 {
		t.Fatalf("probe before linking should be 409, got %d", code)
	}
}
