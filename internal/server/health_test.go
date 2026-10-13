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

// limitedProv is linked but Ford answers 429.
type limitedProv struct{ readyProv }

func (limitedProv) Fetch(context.Context) (*ford.State, error) {
	return nil, &ford.RateLimitError{After: time.Minute}
}

func healthServer(t *testing.T, prov ford.Provider, withLink bool, configured bool) (*httptest.Server, *Server) {
	s, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	cfg := settings.New(s, func(string) (string, string) { return "", "" })
	if configured {
		cfg.Set("FORD_CLIENT_ID", "id")
		cfg.Set("FORD_CLIENT_SECRET", "TOP-SECRET-VALUE")
	}
	d := &trips.Detector{S: s, TZFn: cfg.Location}
	srv := &Server{S: s, D: d, P: poller.New(prov, s, d, poller.Intervals{Parked: time.Minute}), Cfg: cfg, Version: "1.0.0"}
	if withLink {
		srv.Link = reflectLinker{cfg}
	}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts, srv
}

var healthClients = map[string]*http.Client{}

// getHealth reads the status screen as the admin, creating that admin once per server.
func getHealth(t *testing.T, ts *httptest.Server) (string, string) {
	c, ok := healthClients[ts.URL]
	if !ok {
		c = adminClient(t, ts.URL)
		healthClients[ts.URL] = c
	}
	code, body := do(t, c, "GET", ts.URL+"/api/health", "", nil)
	if code != 200 {
		t.Fatalf("health = %d %s", code, body)
	}
	return body, ""
}

func TestHealthNeedsFordSettingsFirst(t *testing.T) {
	ts, _ := healthServer(t, fakeProv{}, true, false)
	body, _ := getHealth(t, ts)
	if !strings.Contains(body, `"overall":"bad"`) || !strings.Contains(body, `"href":"#/admin"`) || !strings.Contains(body, "client ID and secret are not entered") {
		t.Fatalf("unconfigured: %s", body)
	}
}

func TestHealthNotLinkedYet(t *testing.T) {
	ts, _ := healthServer(t, fakeProv{}, true, true)
	body, _ := getHealth(t, ts)
	if !strings.Contains(body, `"href":"#/connect"`) || !strings.Contains(body, "Not linked yet") {
		t.Fatalf("not linked: %s", body)
	}
	if strings.Contains(body, "TOP-SECRET-VALUE") {
		t.Fatalf("health must never include secrets: %s", body)
	}
}

func TestHealthLinkedWaitingThenReading(t *testing.T) {
	ts, srv := healthServer(t, readyProv{}, true, true)
	body, _ := getHealth(t, ts)
	if !strings.Contains(body, "Waiting for the first reading") || !strings.Contains(body, `"overall":"warn"`) {
		t.Fatalf("waiting: %s", body)
	}
	soc := 70.0
	if _, err := srv.S.SaveState("test", &ford.State{TS: time.Now(), SoCPct: &soc, Raw: []byte(`{}`)}); err != nil {
		t.Fatal(err)
	}
	body, _ = getHealth(t, ts)
	if !strings.Contains(body, "Last reading just now") || !strings.Contains(body, `"overall":"ok"`) {
		t.Fatalf("fresh reading: %s", body)
	}
	// An old reading is called out, not hidden.
	if _, err := srv.S.SaveState("test", &ford.State{TS: time.Now().Add(-5 * time.Hour), SoCPct: &soc, Raw: []byte(`{}`)}); err != nil {
		t.Fatal(err)
	}
	srv.S.DB.Exec(`DELETE FROM samples WHERE ts > ?`, time.Now().Add(-time.Hour).Unix())
	body, _ = getHealth(t, ts)
	if !strings.Contains(body, "5 h ago") || !strings.Contains(body, "asleep or out of coverage") {
		t.Fatalf("stale reading: %s", body)
	}
}

func TestHealthExplainsRateLimiting(t *testing.T) {
	ts, srv := healthServer(t, limitedProv{}, true, true)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go srv.P.Run(ctx)
	for i := 0; i < 100 && !srv.P.Status().RateLimited; i++ {
		time.Sleep(50 * time.Millisecond)
	}
	if !srv.P.Status().RateLimited {
		t.Fatal("poller never reported the rate limit")
	}
	body, _ := getHealth(t, ts)
	if !strings.Contains(body, "Ford is limiting how often") || !strings.Contains(body, "Nothing to do") {
		t.Fatalf("rate limit not explained: %s", body)
	}
}

func TestHealthIsReadableByViewersToo(t *testing.T) {
	ts, _ := healthServer(t, readyProv{}, true, true)
	c := adminClient(t, ts.URL)
	do(t, c, "POST", ts.URL+"/api/users", `{"name":"fam","password":"viewer-password-1","role":"viewer"}`, xh)
	vj, _ := cookieJar()
	v := &http.Client{Jar: vj}
	do(t, v, "POST", ts.URL+"/api/login", `{"name":"fam","password":"viewer-password-1"}`, nil)
	if code, _ := do(t, v, "GET", ts.URL+"/api/health", "", nil); code != 200 {
		t.Fatalf("viewer health = %d", code)
	}
	if code, _ := do(t, &http.Client{}, "GET", ts.URL+"/api/health", "", nil); code != 401 {
		t.Fatalf("anonymous health = %d", code)
	}
}
