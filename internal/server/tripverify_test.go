package server

import (
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/ZachCurry13/cartrak/internal/routeest"
	"github.com/ZachCurry13/cartrak/internal/settings"
)

func fakeRoads(t *testing.T, long func(la1, lo1, la2, lo2 float64) bool) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pts := strings.Split(strings.TrimPrefix(r.URL.Path, "/route/v1/driving/"), ";")
		var c [2][2]float64
		for i := 0; i < 2 && i < len(pts); i++ {
			p := strings.Split(pts[i], ",")
			c[i][1], _ = strconv.ParseFloat(p[0], 64) // longitude first
			c[i][0], _ = strconv.ParseFloat(p[1], 64)
		}
		const rad = math.Pi / 180
		a := math.Sin((c[1][0]-c[0][0])*rad/2)*math.Sin((c[1][0]-c[0][0])*rad/2) +
			math.Cos(c[0][0]*rad)*math.Cos(c[1][0]*rad)*math.Sin((c[1][1]-c[0][1])*rad/2)*math.Sin((c[1][1]-c[0][1])*rad/2)
		km := 2 * 6371 * math.Asin(math.Sqrt(a)) * 1.3
		if long(c[0][0], c[0][1], c[1][0], c[1][1]) {
			km *= 3 // the road goes the long way round
		}
		_, _ = fmt.Fprintf(w, `{"code":"Ok","routes":[{"distance":%.0f,"duration":%.0f}]}`, km*1000, km/95*3600)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestAPlanIsCheckedOnRealRoadsAndMadeAgainWhenALegIsLongerThanItLooked(t *testing.T) {
	ts, srv, admin := chainServers(t, 45.8, 46.6, 47.3)
	// the road between the 45.8 and 47.3 chargers goes round a lake: three times as long as it looks
	roads := fakeRoads(t, func(la1, _, la2, _ float64) bool {
		lo, hi := math.Min(la1, la2), math.Max(la1, la2)
		return math.Abs(lo-45.8) < 0.01 && math.Abs(hi-47.3) < 0.01
	})
	srv.RouteEst = &routeest.Client{BaseURL: roads.URL}
	srv.Cfg.Set(settings.RouteKey, "on")
	_, out := do(t, admin, "POST", ts.URL+"/api/trip-chain", chainBody(``), xh)
	if !strings.Contains(out, `"exact":true`) || !strings.Contains(out, `"roads":true`) {
		t.Fatalf("every leg should have been checked on a real road: %s", out)
	}
	if strings.Contains(out, `"name":"Super 47.3"`) && strings.Contains(out, `"name":"Super 45.8"`) && !strings.Contains(out, `"name":"Super 46.6"`) {
		t.Fatalf("the long way round should have been avoided by stopping at the charger in between: %s", out)
	}
	if !strings.Contains(out, `"name":"Super 46.6"`) {
		t.Fatalf("expected the plan to go through the charger in between: %s", out)
	}
}

func TestWithoutRoadEstimatesThePlanSaysItsDistancesAreEstimates(t *testing.T) {
	ts, _, admin := chainServers(t)
	_, out := do(t, admin, "POST", ts.URL+"/api/trip-chain", chainBody(``), xh)
	if !strings.Contains(out, `"exact":false`) || !strings.Contains(out, `"roads":false`) {
		t.Fatalf("with the road lookup off the legs are estimates: %s", out)
	}
}
