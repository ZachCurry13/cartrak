package geocode

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestStateComesWithTheCityAndIsRemembered(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.Write([]byte(`{"address":{"city":"Oakdale","state":"Exampleland","ISO3166-2-lvl4":"US-MN"}}`))
	}))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, UserAgent: "CarTrak/1", Cache: memCache{}, MinGap: time.Millisecond}
	if city, _ := c.City(context.Background(), 12.3, 23.4); city != "Oakdale" {
		t.Fatalf("city %q", city)
	}
	for i := 0; i < 2; i++ {
		if st, err := c.State(context.Background(), 12.3, 23.4); err != nil || st != "MN" {
			t.Fatalf("state %q %v", st, err)
		}
	}
	if hits != 1 {
		t.Fatalf("one request should give both the city and the state, asked %d times", hits)
	}
}

func TestStateIsBlankOutsideTheUS(t *testing.T) {
	if got := pickState(map[string]string{"ISO3166-2-lvl4": "CA-ON"}); got != "" {
		t.Fatalf("an Ontario place is not a US state: %q", got)
	}
	if got := pickState(map[string]string{"state": "Exampleland"}); got != "" {
		t.Fatalf("no ISO code means no state: %q", got)
	}
}
