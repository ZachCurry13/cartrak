package routeest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestARouteIsReadFromTheServiceAndPointsGoLongitudeFirst(t *testing.T) {
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.Write([]byte(`{"code":"Ok","routes":[{"distance":12500,"geometry":{"coordinates":[[-85.1,44.1],[-85.2,44.2],[-85.3,44.3]]}}]}`))
	}))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL}
	pts, km, err := c.Route(context.Background(), []Point{{44.1, -85.1}, {44.3, -85.3}})
	if err != nil || len(pts) != 3 || km != 12.5 || pts[0] != (Point{44.1, -85.1}) {
		t.Fatalf("%v %v %v", pts, km, err)
	}
	if !strings.Contains(path, "-85.10000,44.10000;-85.30000,44.30000") {
		t.Fatalf("longitude goes first: %s", path)
	}
}

func TestABadAnswerIsAnErrorAndImplausibleLengthsAreRejected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"code":"NoRoute","routes":[]}`)) }))
	defer srv.Close()
	if _, _, err := (&Client{BaseURL: srv.URL}).Route(context.Background(), []Point{{1, 1}, {2, 2}}); err == nil {
		t.Fatal("no route is an error")
	}
	if !Plausible(21, 20) || Plausible(50, 20) || Plausible(5, 20) || Plausible(10, 0.2) {
		t.Fatal("length check")
	}
	v := Via([]Point{{0, 0}, {1, 1}, {2, 2}, {3, 3}, {4, 4}, {5, 5}}, 3)
	if len(v) != 3 || v[0] != (Point{0, 0}) || v[2] != (Point{5, 5}) {
		t.Fatalf("via keeps the ends: %v", v)
	}
}
