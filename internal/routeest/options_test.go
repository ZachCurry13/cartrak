package routeest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Every stretch has three roads: 10 km (quickest), 8 km (shortest) and 12 km.
func threeRoads() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"code":"Ok","routes":[
			{"distance":10000,"geometry":{"coordinates":[[-85.1,44.1],[-85.2,44.2]]}},
			{"distance":8000,"geometry":{"coordinates":[[-85.1,44.1],[-85.15,44.15],[-85.2,44.2]]}},
			{"distance":12000,"geometry":{"coordinates":[[-85.1,44.1],[-85.12,44.18],[-85.2,44.2]]}}]}`))
	}))
}

func TestOptionsOfferTheQuickestTheShortestAndTheRoadClosestToTheOdometer(t *testing.T) {
	srv := threeRoads()
	defer srv.Close()
	via := []Point{{44.1, -85.1}, {44.2, -85.2}, {44.3, -85.3}, {44.4, -85.4}} // three stretches
	opts, err := (&Client{BaseURL: srv.URL}).Options(context.Background(), via, 28)
	if err != nil || len(opts) != 3 {
		t.Fatalf("%v %+v", err, opts)
	}
	byKind := map[string]Option{}
	for _, o := range opts {
		byKind[o.Kind] = o
	}
	if byKind["closest"].Km != 28 || byKind["quickest"].Km != 30 || byKind["shortest"].Km != 24 {
		t.Fatalf("10+10+8 is 28, the quickest 30 and the shortest 24: %+v", byKind)
	}
	if opts[0].Kind != "closest" || len(byKind["quickest"].Points) != 4 || len(byKind["shortest"].Points) != 7 {
		t.Fatalf("the closest comes first, and the stretches join without repeating their shared place: %d %d", len(byKind["quickest"].Points), len(byKind["shortest"].Points))
	}
}

func TestOptionsWithoutAnOdometerLengthHaveNoClosestAndSameLengthRoadsAreOneOption(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"code":"Ok","routes":[{"distance":5000,"geometry":{"coordinates":[[-85.1,44.1],[-85.2,44.2]]}}]}`))
	}))
	defer srv.Close()
	opts, err := (&Client{BaseURL: srv.URL}).Options(context.Background(), []Point{{44.1, -85.1}, {44.2, -85.2}}, 0)
	if err != nil || len(opts) != 1 || opts[0].Kind != "quickest" || opts[0].Km != 5 {
		t.Fatalf("%v %+v", err, opts)
	}
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"code":"NoRoute","routes":[]}`)) }))
	defer bad.Close()
	if _, err := (&Client{BaseURL: bad.URL}).Options(context.Background(), []Point{{1, 1}, {2, 2}}, 5); err == nil {
		t.Fatal("no route is an error")
	}
}
