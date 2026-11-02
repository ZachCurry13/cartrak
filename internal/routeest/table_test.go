package routeest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTableGivesMinutesAndKilometersBetweenEveryPairAndRefusesGaps(t *testing.T) {
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.Write([]byte(`{"code":"Ok","durations":[[0,600,1200],[660,0,300],[1260,360,0]],"distances":[[0,5000,12000],[5500,0,3000],[12500,3500,0]]}`))
	}))
	defer srv.Close()
	min, km, err := (&Client{BaseURL: srv.URL}).Table(context.Background(), []Point{{44.1, -85.1}, {44.2, -85.2}, {44.3, -85.3}})
	if err != nil || min[0][1] != 10 || min[2][0] != 21 || km[1][2] != 3 || !strings.Contains(path, "-85.10000,44.10000;-85.20000,44.20000;-85.30000,44.30000") {
		t.Fatalf("%v %v %v %s", min, km, err, path)
	}
	gap := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"code":"Ok","durations":[[0,null],[60,0]],"distances":[[0,null],[1000,0]]}`))
	}))
	defer gap.Close()
	if _, _, err := (&Client{BaseURL: gap.URL}).Table(context.Background(), []Point{{1, 1}, {2, 2}}); err == nil {
		t.Fatal("an unreachable pair is an error, not a free trip")
	}
	if _, _, err := (&Client{}).Table(context.Background(), []Point{{1, 1}}); err == nil {
		t.Fatal("one point has nothing to connect")
	}
}
