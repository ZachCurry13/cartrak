package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestYAMLThenEnvPrecedence(t *testing.T) {
	dir := t.TempDir()
	yml := "CARTRAK_PROVIDER: demo\nFORD_CLIENT_ID: fromfile\nCARTRAK_POLL_PARKED_SEC: 90\nCARTRAK_TZ: America/Chicago\n"
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(yml), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CARTRAK_CONFIG", dir)
	t.Setenv("FORD_CLIENT_ID", "fromenv")
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.Provider != "demo" || c.FordClientID != "fromenv" || c.PollParked.Seconds() != 90 || c.TZ.String() != "America/Chicago" {
		t.Fatalf("bad merge: %+v", c)
	}
	if c.PollDriving.Seconds() != 30 {
		t.Fatalf("default lost: %v", c.PollDriving)
	}
	if _, err := os.Stat(filepath.Join(dir, "config.example.yaml")); err != nil {
		t.Fatal("example not written")
	}
}

func TestMissingFileUsesDefaults(t *testing.T) {
	t.Setenv("CARTRAK_CONFIG", t.TempDir())
	c, err := Load()
	if err != nil || c.Provider != "fordconnect" || c.Addr != ":8080" {
		t.Fatalf("%v %+v", err, c)
	}
}

func TestBadYAMLIsAnError(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("a: [unclosed"), 0o600)
	t.Setenv("CARTRAK_CONFIG", dir)
	if _, err := Load(); err == nil {
		t.Fatal("expected error")
	}
}

func TestProxyAndSessionDefaults(t *testing.T) {
	t.Setenv("CARTRAK_CONFIG", t.TempDir())
	c, _ := Load()
	if !c.TrustProxy || c.SessionDays != 30 {
		t.Fatalf("defaults: %+v", c)
	}
	t.Setenv("CARTRAK_TRUST_PROXY", "false")
	t.Setenv("CARTRAK_SESSION_DAYS", "7")
	c, _ = Load()
	if c.TrustProxy || c.SessionDays != 7 {
		t.Fatalf("overrides: %+v", c)
	}
}

func TestPollIntervalsCannotGoBelowFordsLimit(t *testing.T) {
	t.Setenv("CARTRAK_CONFIG", t.TempDir())
	t.Setenv("CARTRAK_POLL_DRIVING_SEC", "5")
	t.Setenv("CARTRAK_POLL_LIVE_SEC", "2")
	t.Setenv("CARTRAK_POLL_CHARGING_SEC", "10")
	t.Setenv("CARTRAK_POLL_PARKED_SEC", "20")
	c, _ := Load()
	if c.PollDriving.Seconds() != 30 || c.PollLive.Seconds() != 30 || c.PollCharge.Seconds() != 30 || c.PollParked.Seconds() != 60 {
		t.Fatalf("floors not applied: driving=%v live=%v charging=%v parked=%v", c.PollDriving, c.PollLive, c.PollCharge, c.PollParked)
	}
	t.Setenv("CARTRAK_POLL_LIVE_SEC", "")
	c, _ = Load()
	if c.PollLive.Seconds() != 30 {
		t.Fatalf("live default should be 30 s, got %v", c.PollLive)
	}
}
