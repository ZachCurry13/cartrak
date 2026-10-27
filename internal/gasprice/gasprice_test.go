package gasprice

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type memKV map[string]string

func (m memKV) GetKV(k string) string   { return m[k] }
func (m memKV) SetKV(k, v string) error { m[k] = v; return nil }

const body = `{"response":{"data":[{"period":"2026-10-12","duoarea":"SMN","value":"3.400"},{"period":"2026-10-19","duoarea":"SMN","value":3.52},{"period":"2026-10-26","duoarea":"SMN","value":"0"}]}}`

func fake(t *testing.T, gotKey *string, status int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*gotKey = r.URL.Query().Get("api_key")
		if r.URL.Query().Get("facets[duoarea][]") != "SMN" || r.URL.Query().Get("facets[product][]") != "EPMR" {
			t.Errorf("wrong area or product: %s", r.URL.RawQuery)
		}
		w.WriteHeader(status)
		w.Write([]byte(body))
	}))
}

func TestStatesUseTheirOwnPriceAndOthersTheirRegion(t *testing.T) {
	for st, want := range map[string]string{"MN": "Minnesota average", "MI": "Midwest average", "me": "New England average", "TX": "Texas average", "AL": "Gulf Coast average", "": "US average", "ZZ": "US average"} {
		if got := ForState(st).Label; got != want {
			t.Errorf("%q: %q, want %q", st, got, want)
		}
	}
}

func TestPricesAreFetchedRememberedAndPickedByWeek(t *testing.T) {
	var key string
	srv := fake(t, &key, 200)
	defer srv.Close()
	kv := memKV{}
	s := &Service{KV: kv, BaseURL: srv.URL, Now: func() time.Time { return time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC) }}
	a := ForState("MN")
	if err := s.Refresh(context.Background(), "secret-key", a); err != nil || key != "secret-key" {
		t.Fatalf("refresh: %v key %q", err, key)
	}
	day := func(d string) int64 { x, _ := time.Parse("2006-01-02", d); return x.Add(12 * time.Hour).Unix() }
	if p, ok := s.At("secret-key", a, day("2026-10-22")); !ok || p.PerGallon != 3.52 || p.Week != "2026-10-19" || p.Area != "Minnesota average" {
		t.Fatalf("a Thursday uses that week's Monday price: %+v %v", p, ok)
	}
	if p, ok := s.At("secret-key", a, day("2026-10-27")); !ok || p.PerGallon != 3.52 {
		t.Fatalf("a zero (no price) is ignored, so the latest real week stands: %+v", p)
	}
	if _, ok := s.At("secret-key", a, day("2020-01-01")); ok {
		t.Fatal("a day before the first remembered week has no price")
	}
	if _, ok := s.At("", a, day("2026-10-22")); ok {
		t.Fatal("no key means no live price")
	}
}

func TestAnErrorNeverContainsTheKey(t *testing.T) {
	var key string
	srv := fake(t, &key, 500)
	srv.Close() // unreachable
	_, err := Fetch(context.Background(), nil, srv.URL, "secret-key", ForState("MN"), time.Now())
	if err == nil || strings.Contains(err.Error(), "secret-key") {
		t.Fatalf("error must exist and hide the key: %v", err)
	}
	srv2 := fake(t, &key, 403)
	defer srv2.Close()
	if _, err := Fetch(context.Background(), nil, srv2.URL, "secret-key", ForState("MN"), time.Now()); err == nil || strings.Contains(err.Error(), "secret-key") {
		t.Fatalf("refused key: %v", err)
	}
}
