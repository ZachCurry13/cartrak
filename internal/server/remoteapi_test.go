package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/poller"
	"github.com/ZachCurry13/cartrak/internal/settings"
	"github.com/ZachCurry13/cartrak/internal/store"
	"github.com/ZachCurry13/cartrak/internal/trips"
	"github.com/ZachCurry13/cartrak/internal/tunnel"
)

const testTunnelToken = "eyJhIjoiYWJjZGVmMTIzNDU2Nzg5MGFiY2RlZjEyMzQ1Njc4OTAiLCJ0IjoiMTIzNCIsInMiOiJzZWNyZXRzZWNyZXQifQ=="

func remoteServer(t *testing.T, env map[string]string) (*httptest.Server, *Server) {
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
	tun := tunnel.New()
	tun.Bin = "definitely-not-installed-cloudflared"
	t.Cleanup(tun.Stop)
	srv := &Server{S: s, D: d, P: poller.New(fakeProv{}, s, d, poller.Intervals{}), Cfg: cfg, Tunnel: tun, Port: "8080"}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts, srv
}

func TestRemoteSaveExtractsTokenAndHidesIt(t *testing.T) {
	ts, srv := remoteServer(t, nil)
	c := adminClient(t, ts.URL)

	cmd := `sudo cloudflared service install ` + testTunnelToken
	body := `{"token":"` + cmd + `","address":"https://CarTrak.Example.com/","enabled":true}`
	code, resp := do(t, c, "PUT", ts.URL+"/api/admin/remote", body, xh)
	if code != 200 {
		t.Fatalf("save = %d %s", code, resp)
	}
	if strings.Contains(resp, "eyJhIjoi") {
		t.Fatalf("token leaked in the save response: %s", resp)
	}
	for _, want := range []string{`"tokenSet":true`, `"address":"cartrak.example.com"`, `"enabled":true`, `"url":"https://cartrak.example.com"`,
		`"fordRedirect":"https://cartrak.example.com/ford/callback"`, `"port":"8080"`} {
		if !strings.Contains(resp, want) {
			t.Errorf("response missing %s: %s", want, resp)
		}
	}
	if srv.Cfg.Value(keyToken) != testTunnelToken {
		t.Fatal("only the token itself should be stored, not the whole pasted command")
	}
	// The connector program is missing in this test, and the app says so plainly.
	for i := 0; i < 100 && srv.Tunnel.Status().State != tunnel.NotInstalled; i++ {
		time.Sleep(20 * time.Millisecond)
	}
	if _, resp := do(t, c, "GET", ts.URL+"/api/admin/remote", "", nil); !strings.Contains(resp, "not-installed") || strings.Contains(resp, "eyJhIjoi") {
		t.Fatalf("get: %s", resp)
	}
	// The Ford redirect suggestion now uses the public address.
	if _, resp := do(t, c, "GET", ts.URL+"/api/admin/settings", "", nil); !strings.Contains(resp, `"suggestedRedirect":"https://cartrak.example.com/ford/callback"`) {
		t.Fatalf("suggested redirect should use the public address: %s", resp)
	}
	// The generic settings list does not repeat the remote keys.
	if _, resp := do(t, c, "GET", ts.URL+"/api/admin/settings", "", nil); strings.Contains(resp, "CARTRAK_TUNNEL_TOKEN") {
		t.Fatalf("remote keys should not be in the generic list: %s", resp)
	}
}

func TestRemoteValidation(t *testing.T) {
	ts, _ := remoteServer(t, nil)
	c := adminClient(t, ts.URL)
	cases := []struct{ body, why string }{
		{`{"enabled":true}`, "on without a token"},
		{`{"token":"nope"}`, "token that is not a token"},
		{`{"address":"not a host"}`, "bad address"},
		{`{"address":"localhost"}`, "address without a dot"},
	}
	for _, tc := range cases {
		if code, resp := do(t, c, "PUT", ts.URL+"/api/admin/remote", tc.body, xh); code != 400 {
			t.Errorf("%s: want 400, got %d %s", tc.why, code, resp)
		}
	}
	// Saving only an address is fine, and clearing it too.
	if code, _ := do(t, c, "PUT", ts.URL+"/api/admin/remote", `{"address":"a.example.com"}`, xh); code != 200 {
		t.Fatalf("address only = %d", code)
	}
	if code, resp := do(t, c, "PUT", ts.URL+"/api/admin/remote", `{"address":""}`, xh); code != 200 || !strings.Contains(resp, `"address":""`) {
		t.Fatalf("clear address = %d %s", code, resp)
	}
}

func TestRemoteTurnOffAndClearToken(t *testing.T) {
	ts, srv := remoteServer(t, nil)
	c := adminClient(t, ts.URL)
	do(t, c, "PUT", ts.URL+"/api/admin/remote", `{"token":"`+testTunnelToken+`","enabled":true}`, xh)
	// Clearing the token while it is still on is refused; turn it off at the same time.
	if code, _ := do(t, c, "PUT", ts.URL+"/api/admin/remote", `{"clearToken":true}`, xh); code != 400 {
		t.Fatalf("clearing a token that is in use = %d", code)
	}
	if code, _ := do(t, c, "PUT", ts.URL+"/api/admin/remote", `{"clearToken":true,"enabled":false}`, xh); code != 200 {
		t.Fatalf("off and clear = %d", code)
	}
	if srv.Cfg.Value(keyToken) != "" || srv.Tunnel.Status().State != tunnel.Off {
		t.Fatalf("token/state after clear: %q %v", srv.Cfg.Value(keyToken), srv.Tunnel.Status())
	}
}

func TestRemoteLockedAndAdminOnly(t *testing.T) {
	ts, _ := remoteServer(t, map[string]string{keyAddress: "env.example.com"})
	c := adminClient(t, ts.URL)
	if code, resp := do(t, c, "PUT", ts.URL+"/api/admin/remote", `{"address":"other.example.com"}`, xh); code != 409 || !strings.Contains(resp, "config.yaml") {
		t.Fatalf("locked address changed: %d %s", code, resp)
	}
	if _, resp := do(t, c, "GET", ts.URL+"/api/admin/remote", "", nil); !strings.Contains(resp, `"address":"env.example.com"`) || !strings.Contains(resp, `"locked":{"token":false,"address":true`) {
		t.Fatalf("locked flag missing: %s", resp)
	}

	do(t, c, "POST", ts.URL+"/api/users", `{"name":"fam","password":"viewer-password-1","role":"viewer"}`, xh)
	vj, _ := cookieJar()
	v := &http.Client{Jar: vj}
	do(t, v, "POST", ts.URL+"/api/login", `{"name":"fam","password":"viewer-password-1"}`, nil)
	if code, _ := do(t, v, "GET", ts.URL+"/api/admin/remote", "", nil); code != 403 {
		t.Fatalf("viewer read remote settings = %d", code)
	}
	if code, _ := do(t, v, "PUT", ts.URL+"/api/admin/remote", `{"enabled":false}`, xh); code != 403 {
		t.Fatalf("viewer changed remote settings = %d", code)
	}
	anon := &http.Client{}
	if code, _ := do(t, anon, "GET", ts.URL+"/api/admin/remote", "", nil); code != 401 {
		t.Fatalf("anonymous = %d", code)
	}
}
