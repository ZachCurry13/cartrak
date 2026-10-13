package geocode

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

type memCache map[string]string

func (m memCache) GetPlace(k string) (string, bool) { v, ok := m[k]; return v, ok }
func (m memCache) SetPlace(k, v string) error       { m[k] = v; return nil }

func TestCityIsLookedUpOnceAndRoundedBeforeItLeaves(t *testing.T) {
	var hits int32
	var gotLat, gotLon, ua string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		gotLat, gotLon, ua = r.URL.Query().Get("lat"), r.URL.Query().Get("lon"), r.Header.Get("User-Agent")
		w.Write([]byte(`{"address":{"city":"Oakdale","county":"Example County","state":"Exampleland"}}`))
	}))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, UserAgent: "CarTrak/1 (+example)", Cache: memCache{}, MinGap: time.Millisecond}

	for i := 0; i < 3; i++ {
		city, err := c.City(context.Background(), 12.3456789, 23.4567890)
		if err != nil || city != "Oakdale" {
			t.Fatalf("lookup %d: %q %v", i, city, err)
		}
	}
	if hits != 1 {
		t.Fatalf("asked the service %d times; the answer should be remembered", hits)
	}
	if gotLat != "12.346" || gotLon != "23.457" {
		t.Fatalf("locations must be rounded to about 100 m before they leave: %s,%s", gotLat, gotLon)
	}
	if ua != "CarTrak/1 (+example)" {
		t.Fatalf("an identifying User-Agent is required by the service: %q", ua)
	}
	// A nearby point rounds to the same place, so it is not looked up again.
	if _, err := c.City(context.Background(), 12.3459, 23.4571); err != nil || hits != 1 {
		t.Fatalf("a point 30 m away should reuse the answer (hits %d, %v)", hits, err)
	}
}

func TestNamesAreChosenFromTheMostSpecificSettlement(t *testing.T) {
	for _, c := range []struct {
		a    map[string]string
		want string
	}{
		{map[string]string{"town": "Springfield", "county": "Sample County"}, "Springfield"},
		{map[string]string{"village": "Hillcrest", "county": "Example County"}, "Hillcrest"},
		{map[string]string{"city": "Port Lakeside", "suburb": "x"}, "Port Lakeside"},
		{map[string]string{"county": "Sample County"}, "Sample County"},
		{map[string]string{"state": "Exampleland"}, ""},
	} {
		if got := pickCity(c.a); got != c.want {
			t.Errorf("%v: got %q, want %q", c.a, got, c.want)
		}
	}
}

func TestFailuresAreNotCachedAndRequestsAreSpacedOut(t *testing.T) {
	var n int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&n, 1) == 1 {
			w.WriteHeader(429)
			return
		}
		w.Write([]byte(`{"address":{"city":"Rivertown"}}`))
	}))
	defer srv.Close()
	cache := memCache{}
	c := &Client{BaseURL: srv.URL, Cache: cache, MinGap: 60 * time.Millisecond}
	if _, err := c.City(context.Background(), 12.1, 23.5); err == nil {
		t.Fatal("an HTTP 429 must be reported")
	}
	if len(cache) != 0 {
		t.Fatal("a failed lookup must not be remembered")
	}
	start := time.Now()
	if city, err := c.City(context.Background(), 12.1, 23.5); err != nil || city != "Rivertown" {
		t.Fatalf("retry: %q %v", city, err)
	}
	if time.Since(start) < 30*time.Millisecond {
		t.Fatal("requests to the service must be spaced out")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := (&Client{BaseURL: srv.URL, MinGap: time.Hour, last: time.Now()}).City(ctx, 1, 1); err == nil {
		t.Fatal("a cancelled request must stop waiting")
	}
}
