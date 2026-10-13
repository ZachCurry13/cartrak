package connector

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/tunnel"
)

// The test program doubles as a stand-in for cloudflared: the supervisor and the installer run it.
func TestMain(m *testing.M) {
	if len(os.Args) == 2 && os.Args[1] == "--version" {
		v := os.Getenv("FAKE_CF_OLD")
		if strings.Contains(os.Args[0], "cloudflared-") || strings.Contains(os.Args[0], "download.part") {
			v = os.Getenv("FAKE_CF_NEW")
		}
		fmt.Println("cloudflared version " + v + " (built today)")
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("FAKE_CLOUDFLARED") == "" {
		return
	}
	if os.Getenv("FAKE_NEW_CRASHES") == "1" && (strings.Contains(os.Args[0], "cloudflared-") || strings.Contains(os.Args[0], "download.part")) {
		fmt.Println("ERR cannot reach Cloudflare")
		os.Exit(3)
	}
	fmt.Println("INF Registered tunnel connection connIndex=0")
	time.Sleep(30 * time.Second)
}

type kv struct {
	mu sync.Mutex
	m  map[string]string
}

func (k *kv) GetKV(a string) string { k.mu.Lock(); defer k.mu.Unlock(); return k.m[a] }
func (k *kv) SetKV(a, b string) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.m[a] = b
	return nil
}

// fakeGitHub serves a release whose Linux file is this very test program.
func fakeGitHub(t *testing.T, version, digest string) *httptest.Server {
	exe, _ := os.ReadFile(os.Args[0])
	if digest == "match" {
		sum := sha256.Sum256(exe)
		digest = "sha256:" + hex.EncodeToString(sum[:])
	}
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/file" {
			w.Write(exe)
			return
		}
		fmt.Fprintf(w, `{"tag_name":"%s","assets":[{"name":"cloudflared-linux-amd64","browser_download_url":"%s/file","digest":"%s"},{"name":"cloudflared-linux-arm64","browser_download_url":"%s/nope","digest":"sha256:00"}]}`, version, srv.URL, digest, srv.URL)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func setup(t *testing.T, srv *httptest.Server, running bool) (*Manager, *tunnel.Supervisor, *kv) {
	t.Setenv("FAKE_CF_OLD", "2026.10.0")
	t.Setenv("FAKE_CF_NEW", "2026.10.1")
	t.Setenv("FAKE_CLOUDFLARED", "1")
	dir := t.TempDir() // made first, so it is removed after the connector has stopped
	sup := tunnel.New()
	sup.SetProgram(os.Args[0])
	sup.Args = []string{"-test.run=TestHelperProcess"}
	t.Cleanup(func() { sup.Stop(); time.Sleep(500 * time.Millisecond) })
	k := &kv{m: map[string]string{}}
	m := &Manager{Sup: sup, Dir: filepath.Join(dir, "bin"), KV: k, API: srv.URL, GOOS: "linux", GOARCH: "amd64",
		AllowInsecure: true, AllowHost: func(string) bool { return true }, Grace: 5 * time.Second}
	if runtime.GOOS == "windows" {
		m.ExeSuffix = ".exe"
	}
	m.Restore()
	if running {
		sup.Apply(true, "eyJhIjoiYWJjZGVmMTIzNDU2Nzg5MGFiY2RlZjEyMzQ1Njc4OTAiLCJ0IjoiMTIzNCIsInMiOiJzZWNyZXRzZWNyZXQifQ==")
		if !sup.WaitConnected(5 * time.Second) {
			t.Fatal("the old connector never connected")
		}
	}
	return m, sup, k
}

func TestANewerConnectorIsOfferedAndInstalledWithTheTunnelStayingUp(t *testing.T) {
	m, sup, k := setup(t, fakeGitHub(t, "2026.10.1", "match"), true)
	if err := m.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	if i := m.Info(); i.Current != "2026.10.0" || i.Latest != "2026.10.1" || !i.Available || !i.Supported {
		t.Fatalf("%+v", i)
	}
	if err := m.Update(context.Background()); err != nil {
		t.Fatalf("update: %v (%+v)", err, m.Info().Update)
	}
	i := m.Info()
	if i.Current != "2026.10.1" || i.Available || i.Update.State != "done" || !strings.Contains(sup.Program(), "cloudflared-2026.10.1") {
		t.Fatalf("%+v program %s", i, sup.Program())
	}
	if k.GetKV(kvBin) != sup.Program() || k.GetKV(kvPrev) != os.Args[0] {
		t.Fatalf("remembered for the next start: %v", k.m)
	}
	if sup.Status().State != tunnel.Connected {
		t.Fatalf("the tunnel is connected after the switch: %+v", sup.Status())
	}
}

func TestANewConnectorThatDoesNotConnectIsRolledBack(t *testing.T) {
	t.Setenv("FAKE_NEW_CRASHES", "1")
	m, sup, k := setup(t, fakeGitHub(t, "2026.10.1", "match"), true)
	m.Grace = 3 * time.Second
	_ = m.Check(context.Background())
	if err := m.Update(context.Background()); err == nil {
		t.Fatal("the update must fail")
	}
	if sup.Program() != os.Args[0] || k.GetKV(kvBin) != "" || k.GetKV(kvFailed) != "2026.10.1" {
		t.Fatalf("the old one is back and the failed version is remembered: %s %v", sup.Program(), k.m)
	}
	if !sup.WaitConnected(5*time.Second) || m.Info().Available || m.Info().Failed != "2026.10.1" || m.Info().Update.State != "failed" {
		t.Fatalf("connected again with the old one, and not offered again: %+v %+v", sup.Status(), m.Info())
	}
	if files, _ := filepath.Glob(filepath.Join(m.Dir, "cloudflared-*")); len(files) != 0 {
		t.Fatalf("the failed file is removed: %v", files)
	}
}

func TestADownloadThatDoesNotMatchItsChecksumOrHasNoneIsRefused(t *testing.T) {
	m, sup, _ := setup(t, fakeGitHub(t, "2026.10.1", "sha256:"+strings.Repeat("ab", 32)), false)
	_ = m.Check(context.Background())
	if err := m.Update(context.Background()); err == nil || !strings.Contains(m.Info().Update.Message, "checksum") || sup.Program() != os.Args[0] {
		t.Fatalf("wrong checksum: %v %+v", err, m.Info().Update)
	}
	m2, _, _ := setup(t, fakeGitHub(t, "2026.10.1", ""), false)
	_ = m2.Check(context.Background())
	if err := m2.Update(context.Background()); err == nil || !strings.Contains(m2.Info().Update.Message, "did not publish a checksum") {
		t.Fatalf("no checksum: %v %+v", err, m2.Info().Update)
	}
}

func TestOnlyGitHubHostsAndNewerVersionsAreAccepted(t *testing.T) {
	m := &Manager{}
	for _, bad := range []string{"http://github.com/x", "https://evil.example/x", "https://github.com.evil.example/x"} {
		u, _ := urlParse(bad)
		if m.checkURL(u) == nil {
			t.Errorf("%s should be refused", bad)
		}
	}
	for _, good := range []string{"https://github.com/cloudflare/cloudflared/releases/download/x", "https://objects.githubusercontent.com/x", "https://release-assets.githubusercontent.com/x"} {
		u, _ := urlParse(good)
		if err := m.checkURL(u); err != nil {
			t.Errorf("%s: %v", good, err)
		}
	}
	for _, c := range []struct {
		a, b string
		want bool
	}{{"2026.10.1", "2026.10.0", true}, {"2026.10.0", "2026.10.0", false}, {"2026.9.5", "2026.10.0", false}, {"2027.1.0", "2026.12.9", true}} {
		if Newer(c.a, c.b) != c.want {
			t.Errorf("Newer(%s, %s) should be %v", c.a, c.b, c.want)
		}
	}
}

func urlParse(s string) (*url.URL, error) { return url.Parse(s) }

func TestAnInstalledConnectorIsUsedAtStartUnlessTheImageNowShipsANewerOne(t *testing.T) {
	m, sup, k := setup(t, fakeGitHub(t, "2026.10.1", "match"), false)
	_ = m.Check(context.Background())
	if err := m.Update(context.Background()); err != nil {
		t.Fatal(err)
	}
	installed := sup.Program()
	sup.SetProgram(os.Args[0]) // a restart: the bundled connector is the default again
	m.Restore()
	if sup.Program() != installed || m.Info().Current != "2026.10.1" {
		t.Fatalf("the installed one is used again: %s %+v", sup.Program(), m.Info())
	}
	t.Setenv("FAKE_CF_OLD", "2026.11.0") // the image was updated and now ships a newer connector
	sup.SetProgram(os.Args[0])
	m.Restore()
	if sup.Program() != os.Args[0] || k.GetKV(kvBin) != "" || m.Info().Current != "2026.11.0" {
		t.Fatalf("the bundled one wins: %s %v %+v", sup.Program(), k.m, m.Info())
	}
}
