package routeest

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRouteNearTakesTheAlternativeClosestToTheOdometer(t *testing.T) {
	var asked string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked = r.URL.RawQuery
		route := func(m int) string {
			return fmt.Sprintf(`{"distance":%d,"geometry":{"coordinates":[[-93.0,45.0],[-93.%d,45.1]]}}`, m, m/1000)
		}
		fmt.Fprintf(w, `{"code":"Ok","routes":[%s,%s,%s]}`, route(12900), route(4200), route(6000)) // the quickest is a long way round
	}))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL}
	_, km, err := c.RouteNear(context.Background(), []Point{{45, -93}, {45.1, -93.1}}, 4.0)
	if err != nil || km < 4.1 || km > 4.3 || !strings.Contains(asked, "alternatives=3") {
		t.Fatalf("the 4.2 km road, asked for alternatives: %v %v %q", km, err, asked)
	}
	if _, km, _ = c.RouteNear(context.Background(), []Point{{45, -93}, {45.1, -93.1}}, 0); km < 12.8 {
		t.Fatalf("no target takes the first answer: %v", km)
	}
}
