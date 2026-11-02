package geocode

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSearchReturnsMatchesAndSkipsBadOnes(t *testing.T) {
	var gotQ, gotAgent string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQ, gotAgent = r.URL.Query().Get("q"), r.Header.Get("User-Agent")
		w.Write([]byte(`[{"lat":"44.97","lon":"-93.26","display_name":"1 Main St, Townsville"},{"lat":"x","lon":"1","display_name":"bad"},{"lat":"95","lon":"1","display_name":"off the map"}]`))
	}))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, UserAgent: "test", MinGap: time.Millisecond}
	got, err := c.Search(context.Background(), "1 Main St Townsville", nil)
	if err != nil || len(got) != 1 || got[0].Label != "1 Main St, Townsville" || got[0].Lat != 44.97 || got[0].Lon != -93.26 {
		t.Fatalf("got %+v, %v", got, err)
	}
	if gotQ != "1 Main St Townsville" || gotAgent != "test" {
		t.Fatalf("asked %q as %q", gotQ, gotAgent)
	}
}

func TestSearchRejectsAnEmptyOrHugeQueryWithoutAsking(t *testing.T) {
	c := &Client{BaseURL: "http://127.0.0.1:1", MinGap: time.Millisecond}
	for _, q := range []string{"", "  ", "ab", string(make([]byte, 300))} {
		if _, err := c.Search(context.Background(), q, nil); err == nil {
			t.Fatalf("%q was accepted", q)
		}
	}
}

func TestSearchReportsAServiceError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(503) }))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, MinGap: time.Millisecond}
	if _, err := c.Search(context.Background(), "1 Main St", nil); err == nil {
		t.Fatal("a 503 was not an error")
	}
}

func TestSearchPrefersTheRegionAroundTheCarWithoutSendingItsExactPlace(t *testing.T) {
	var query string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.RawQuery
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, MinGap: time.Millisecond}
	near := [2]float64{44.9778, -93.2650}
	if _, err := c.Search(context.Background(), "Salon Innovations", &near); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(query, "viewbox=-96%2C48%2C-90%2C42") || !strings.Contains(query, "bounded=0") || strings.Contains(query, "44.97") || strings.Contains(query, "93.26") {
		t.Fatalf("a box rounded to whole degrees, a preference only: %s", query)
	}
}

func TestSearchNamesTheBusinessGivesAShortAddressAndPutsTheClosestFirst(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[
			{"lat":"46.80","lon":"-92.10","display_name":"Flexible Circuits, 9 Far Rd, Duluth, St. Louis County, Minnesota, United States","name":"Flexible Circuits","category":"office","type":"company","address":{"house_number":"9","road":"Far Rd","city":"Duluth","state":"Minnesota","ISO3166-2-lvl4":"US-MN"}},
			{"lat":"45.07","lon":"-93.46","display_name":"Flexible Circuits, 1200 Main St, Maple Grove, Hennepin County, Minnesota, United States","name":"Flexible Circuits","category":"office","type":"company","address":{"house_number":"1200","road":"Main St","city":"Maple Grove","state":"Minnesota","ISO3166-2-lvl4":"US-MN"}},
			{"lat":"45.0","lon":"-93.4","display_name":"Elm Street, Somewhere","category":"highway","type":"residential","address":{"road":"Elm Street","town":"Somewhere"}}]`))
	}))
	defer srv.Close()
	near := [2]float64{45.0, -93.3}
	got, err := (&Client{BaseURL: srv.URL, MinGap: time.Millisecond}).Search(context.Background(), "flexible circuits", &near)
	if err != nil || len(got) != 3 {
		t.Fatalf("%v %+v", err, got)
	}
	if got[0].Name != "Elm Street" || got[1].Name != "Flexible Circuits" || got[1].Address != "1200 Main St, Maple Grove, MN" || got[1].Kind != "company" || got[2].Address != "9 Far Rd, Duluth, MN" {
		t.Fatalf("closest first, with a short address and the kind of place: %+v", got)
	}
	if got[0].Kind != "" || got[1].DistKm == nil || *got[1].DistKm < 10 || *got[1].DistKm > 40 || *got[2].DistKm < 150 {
		t.Fatalf("a street has no kind, and each carries its distance: %+v %v %v", got[0], *got[1].DistKm, *got[2].DistKm)
	}
	none, _ := (&Client{BaseURL: srv.URL, MinGap: time.Millisecond}).Search(context.Background(), "flexible circuits", nil)
	if none[0].DistKm != nil || none[0].Name != "Flexible Circuits" {
		t.Fatalf("without a place to look near, the order is the service's and there are no distances: %+v", none[0])
	}
}
