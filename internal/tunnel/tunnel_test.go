package tunnel

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

const fakeToken = "eyJhIjoiYWJjZGVmMTIzNDU2Nzg5MGFiY2RlZjEyMzQ1Njc4OTAiLCJ0IjoiMTIzNCIsInMiOiJzZWNyZXRzZWNyZXQifQ=="

// TestHelperProcess is not a real test: the supervisor tests run this test
// binary as a stand-in for cloudflared.
func TestHelperProcess(t *testing.T) {
	mode := os.Getenv("FAKE_CLOUDFLARED")
	if mode == "" {
		return
	}
	fmt.Println("INF Starting tunnel")
	fmt.Println("INF token seen in env:", os.Getenv("TUNNEL_TOKEN"))
	switch mode {
	case "connect":
		fmt.Println("INF Registered tunnel connection connIndex=0")
		time.Sleep(30 * time.Second)
	case "crash":
		fmt.Println("ERR something broke")
		os.Exit(3)
	}
	os.Exit(0)
}

func fake(t *testing.T, mode string) *Supervisor {
	t.Setenv("FAKE_CLOUDFLARED", mode)
	s := New()
	s.Bin = os.Args[0]
	s.Args = []string{"-test.run=TestHelperProcess"}
	s.backoff = 50 * time.Millisecond
	t.Cleanup(s.Stop)
	return s
}

func waitFor(t *testing.T, what string, ok func() bool) {
	t.Helper()
	for i := 0; i < 200; i++ {
		if ok() {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func TestConnectsAndNeverLogsTheToken(t *testing.T) {
	s := fake(t, "connect")
	s.Apply(true, fakeToken)
	waitFor(t, "connected", func() bool { return s.Status().State == Connected })
	logs := strings.Join(s.Log(), "\n")
	if strings.Contains(logs, fakeToken) || strings.Contains(logs, "eyJhIjoi") {
		t.Fatalf("token leaked into the log:\n%s", logs)
	}
	if !strings.Contains(logs, "token seen in env: <token>") {
		t.Fatalf("token should reach the process through its environment and be redacted in the log:\n%s", logs)
	}
	// Applying the same settings again does not restart it.
	since := s.Status().Since
	s.Apply(true, fakeToken)
	time.Sleep(100 * time.Millisecond)
	if s.Status().State != Connected || s.Status().Since != since {
		t.Fatalf("idempotent Apply restarted the connector: %+v", s.Status())
	}
}

func TestTurningOffStopsIt(t *testing.T) {
	s := fake(t, "connect")
	s.Apply(true, fakeToken)
	waitFor(t, "connected", func() bool { return s.Status().State == Connected })
	s.Apply(false, fakeToken)
	if s.Status().State != Off {
		t.Fatalf("want off, got %+v", s.Status())
	}
	s.Apply(true, "") // enabled without a token is also off
	if s.Status().State != Off {
		t.Fatalf("no token should be off: %+v", s.Status())
	}
}

func TestChangingTheTokenRestarts(t *testing.T) {
	s := fake(t, "connect")
	s.Apply(true, fakeToken)
	waitFor(t, "connected", func() bool { return s.Status().State == Connected })
	s.Apply(true, fakeToken+"x")
	if st := s.Status(); st.State != Starting {
		t.Fatalf("new token should restart: %+v", st)
	}
	waitFor(t, "connected again", func() bool { return s.Status().State == Connected })
}

func TestCrashRestartsWithBackoff(t *testing.T) {
	s := fake(t, "crash")
	s.Apply(true, fakeToken)
	waitFor(t, "reconnecting", func() bool { return s.Status().State == Reconnecting })
	if m := s.Status().Message; !strings.Contains(m, "exit code 3") || !strings.Contains(m, "Restarting") {
		t.Fatalf("message: %q", m)
	}
	// It keeps trying: the log grows past a single run.
	waitFor(t, "second attempt", func() bool {
		n := 0
		for _, l := range s.Log() {
			if strings.Contains(l, "Starting tunnel") {
				n++
			}
		}
		return n >= 2
	})
}

func TestNotInstalled(t *testing.T) {
	s := New()
	s.Bin = "definitely-not-a-real-program-xyz"
	t.Cleanup(s.Stop)
	if s.Installed() {
		t.Fatal("should not be installed")
	}
	s.Apply(true, fakeToken)
	waitFor(t, "not installed", func() bool { return s.Status().State == NotInstalled })
	if !strings.Contains(s.Status().Message, "not included") {
		t.Fatalf("message: %q", s.Status().Message)
	}
}

func TestExtractToken(t *testing.T) {
	good := []string{
		fakeToken,
		"  " + fakeToken + "\n",
		"sudo cloudflared service install " + fakeToken,
		"cloudflared.exe service install " + fakeToken,
		"docker run cloudflare/cloudflared:latest tunnel --no-autoupdate run --token " + fakeToken,
	}
	for _, in := range good {
		if got, err := ExtractToken(in); err != nil || got != fakeToken {
			t.Errorf("ExtractToken(%q) = %q, %v", in, got, err)
		}
	}
	for _, in := range []string{"", "short", "hello world this is not a token at all, sorry", "cloudflared tunnel run"} {
		if _, err := ExtractToken(in); err == nil {
			t.Errorf("%q should be rejected", in)
		}
	}
}

func TestNormalizeAddress(t *testing.T) {
	ok := map[string]string{
		"cartrak.example.com":            "cartrak.example.com",
		"https://Cartrak.Example.com/":   "cartrak.example.com",
		"  http://car.example.co.uk/x?y": "car.example.co.uk",
	}
	for in, want := range ok {
		if got, err := NormalizeAddress(in); err != nil || got != want {
			t.Errorf("NormalizeAddress(%q) = %q, %v", in, got, err)
		}
	}
	for _, in := range []string{"", "localhost", "example", "bad host.com", "10.0.0.5", "a..com", "https://"} {
		if _, err := NormalizeAddress(in); err == nil {
			t.Errorf("%q should be rejected", in)
		}
	}
}

func TestRedact(t *testing.T) {
	got := redact("connecting with "+fakeToken+" and eyJzb21ldGhpbmdlbHNlbG9uZ2VyMTIzNDU2Nzg5", fakeToken)
	if strings.Contains(got, "eyJ") || !strings.Contains(got, "<token>") {
		t.Fatalf("not redacted: %s", got)
	}
}
