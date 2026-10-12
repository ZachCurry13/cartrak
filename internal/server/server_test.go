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
	"github.com/ZachCurry13/cartrak/internal/store"
	"github.com/ZachCurry13/cartrak/internal/trips"
)

type fakeProv struct{}

func (fakeProv) Name() string                               { return "fake" }
func (fakeProv) Ready() bool                                { return false }
func (fakeProv) Fetch(context.Context) (*ford.State, error) { return nil, ford.ErrNotLinked }

func newTestServer(t *testing.T) (*httptest.Server, *store.Store) {
	s, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	d := &trips.Detector{S: s, TZ: time.UTC}
	srv := &Server{S: s, D: d, P: poller.New(fakeProv{}, s, d, poller.Intervals{})}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts, s
}

func do(t *testing.T, c *http.Client, method, url, body string, hdr map[string]string) (int, string) {
	req, _ := http.NewRequest(method, url, strings.NewReader(body))
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b := new(strings.Builder)
	buf := make([]byte, 4096)
	for {
		n, e := resp.Body.Read(buf)
		b.Write(buf[:n])
		if e != nil {
			break
		}
	}
	return resp.StatusCode, b.String()
}

func TestAuthFlowAndRoles(t *testing.T) {
	ts, _ := newTestServer(t)
	jar, _ := cookieJar()
	c := &http.Client{Jar: jar}
	x := map[string]string{"X-Requested-With": "cartrak"}

	if code, _ := do(t, c, "GET", ts.URL+"/api/trips", "", nil); code != 401 {
		t.Fatalf("trips without login = %d", code)
	}
	if code, _ := do(t, c, "POST", ts.URL+"/api/setup", `{"name":"a","password":"short"}`, nil); code != 400 {
		t.Fatalf("short password accepted: %d", code)
	}
	if code, _ := do(t, c, "POST", ts.URL+"/api/setup", `{"name":"owner","password":"correct-horse-battery"}`, nil); code != 200 {
		t.Fatalf("setup = %d", code)
	}
	if code, _ := do(t, c, "POST", ts.URL+"/api/setup", `{"name":"x","password":"correct-horse-battery"}`, nil); code != 403 {
		t.Fatalf("second setup allowed: %d", code)
	}
	if code, _ := do(t, c, "GET", ts.URL+"/api/trips", "", nil); code != 200 {
		t.Fatalf("trips after setup = %d", code)
	}
	// CSRF: state-changing calls need the header.
	if code, _ := do(t, c, "POST", ts.URL+"/api/rules", `{"name":"w","category":"work","days":62,"startMin":480,"endMin":1020,"enabled":true}`, nil); code != 403 {
		t.Fatalf("rule without CSRF header = %d", code)
	}
	if code, _ := do(t, c, "POST", ts.URL+"/api/rules", `{"name":"w","category":"work","days":62,"startMin":480,"endMin":1020,"enabled":true}`, x); code != 200 {
		t.Fatalf("rule create = %d", code)
	}
	if code, _ := do(t, c, "POST", ts.URL+"/api/users", `{"name":"fam","password":"viewer-password-1","role":"viewer"}`, x); code != 200 {
		t.Fatalf("add viewer = %d", code)
	}

	// A viewer can read but not change anything.
	vj, _ := cookieJar()
	v := &http.Client{Jar: vj}
	if code, _ := do(t, v, "POST", ts.URL+"/api/login", `{"name":"fam","password":"wrong"}`, nil); code != 401 {
		t.Fatalf("bad login = %d", code)
	}
	if code, _ := do(t, v, "POST", ts.URL+"/api/login", `{"name":"fam","password":"viewer-password-1"}`, nil); code != 200 {
		t.Fatalf("viewer login = %d", code)
	}
	if code, _ := do(t, v, "GET", ts.URL+"/api/rules", "", nil); code != 200 {
		t.Fatalf("viewer read rules = %d", code)
	}
	if code, _ := do(t, v, "POST", ts.URL+"/api/rules", `{}`, x); code != 403 {
		t.Fatalf("viewer wrote rules = %d", code)
	}
	if code, _ := do(t, v, "GET", ts.URL+"/api/users", "", nil); code != 403 {
		t.Fatalf("viewer listed users = %d", code)
	}
}

func TestLoginRateLimit(t *testing.T) {
	ts, s := newTestServer(t)
	h, _ := hashPassword("correct-horse-battery")
	s.CreateUser("owner", h, "admin")
	c := &http.Client{}
	var last int
	for i := 0; i < 7; i++ {
		last, _ = do(t, c, "POST", ts.URL+"/api/login", `{"name":"owner","password":"nope"}`, nil)
	}
	if last != 429 {
		t.Fatalf("expected lockout, got %d", last)
	}
}
