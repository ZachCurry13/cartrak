package settings

import "testing"

type memKV map[string]string

func (m memKV) GetKV(k string) string   { return m[k] }
func (m memKV) SetKV(k, v string) error { m[k] = v; return nil }

func proc(m map[string]string) Lookup {
	return func(k string) (string, string) {
		if v := m[k]; v != "" {
			return v, "env"
		}
		return "", ""
	}
}

func TestPrecedenceAndLocking(t *testing.T) {
	kv := memKV{}
	s := New(kv, proc(map[string]string{"FORD_CLIENT_ID": "from-env"}))
	s.Set("FORD_CLIENT_ID", "from-admin")
	s.Set("FORD_APPLICATION_ID", "app-admin")
	if v, src := s.Get("FORD_CLIENT_ID"); v != "from-env" || src != "env" || !s.Locked("FORD_CLIENT_ID") {
		t.Fatalf("env must win: %v %v", v, src)
	}
	if v, src := s.Get("FORD_APPLICATION_ID"); v != "app-admin" || src != "admin" || s.Locked("FORD_APPLICATION_ID") {
		t.Fatalf("admin value: %v %v", v, src)
	}
	if v, src := s.Get("CARTRAK_TZ"); v != "UTC" || src != "default" {
		t.Fatalf("tz default: %v %v", v, src)
	}
}

func TestDerivedRedirectAndClear(t *testing.T) {
	kv := memKV{"derived_redirect": "https://x.example/ford/callback"}
	s := New(kv, proc(nil))
	if v, src := s.Get("FORD_REDIRECT_URL"); v != "https://x.example/ford/callback" || src != "default" {
		t.Fatalf("%v %v", v, src)
	}
	s.Set("FORD_REDIRECT_URL", "https://y.example/ford/callback")
	if s.Value("FORD_REDIRECT_URL") != "https://y.example/ford/callback" {
		t.Fatal("admin value should beat derived")
	}
	s.Set("FORD_REDIRECT_URL", "")
	if s.Value("FORD_REDIRECT_URL") != "https://x.example/ford/callback" {
		t.Fatal("clearing should fall back to derived")
	}
}

func TestConfiguredAndLocation(t *testing.T) {
	s := New(memKV{}, proc(nil))
	if s.Configured() {
		t.Fatal("empty is not configured")
	}
	s.Set("FORD_CLIENT_ID", "a")
	s.Set("FORD_CLIENT_SECRET", "b")
	s.Set("FORD_APPLICATION_ID", "c")
	s.Set("CARTRAK_TZ", "America/Chicago")
	if !s.Configured() || s.Location().String() != "America/Chicago" {
		t.Fatal("configured/location wrong")
	}
	s.Set("CARTRAK_TZ", "Not/AZone")
	if s.Location().String() != "UTC" {
		t.Fatal("bad zone must fall back to UTC")
	}
}

func TestValid(t *testing.T) {
	bad := [][2]string{{"CARTRAK_TZ", "Mars/Base"}, {"FORD_REDIRECT_URL", "cartrak.example.com/cb"}, {"FORD_REDIRECT_URL", "http://example.com/cb"}}
	for _, b := range bad {
		if Valid(b[0], b[1]) == nil {
			t.Errorf("%v should be invalid", b)
		}
	}
	good := [][2]string{{"CARTRAK_TZ", "Europe/Paris"}, {"FORD_REDIRECT_URL", "https://a.example/ford/callback"}, {"FORD_REDIRECT_URL", "http://localhost:8080/ford/callback"}, {"FORD_VIN", "x"}}
	for _, g := range good {
		if err := Valid(g[0], g[1]); err != nil {
			t.Errorf("%v: %v", g, err)
		}
	}
	if _, ok := Known("FORD_CLIENT_SECRET"); !ok {
		t.Fatal("known key missing")
	}
}
