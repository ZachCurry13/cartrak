package server

import (
	"encoding/json"
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

// pacedRoads answers every leg as one stretch of highway at 100 km/h, a quarter longer than the straight line.
func pacedRoads(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pts := strings.Split(strings.TrimPrefix(r.URL.Path, "/route/v1/driving/"), ";")
		var c [2][2]float64
		for i := 0; i < 2 && i < len(pts); i++ {
			p := strings.Split(pts[i], ",")
			c[i][1], _ = strconv.ParseFloat(p[0], 64)
			c[i][0], _ = strconv.ParseFloat(p[1], 64)
		}
		const rad = math.Pi / 180
		a := math.Sin((c[1][0]-c[0][0])*rad/2)*math.Sin((c[1][0]-c[0][0])*rad/2) +
			math.Cos(c[0][0]*rad)*math.Cos(c[1][0]*rad)*math.Sin((c[1][1]-c[0][1])*rad/2)*math.Sin((c[1][1]-c[0][1])*rad/2)
		km := 2 * 6371 * math.Asin(math.Sqrt(a)) * 1.25
		_, _ = fmt.Fprintf(w, `{"code":"Ok","routes":[{"distance":%.0f,"duration":%.0f,"legs":[{"annotation":{"distance":[%.0f],"speed":[27.78]}}]}]}`, km*1000, km/100*3600, km*1000)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestDrivingFasterOnHighwaysShortensTheDriveByThatPace(t *testing.T) {
	ts, srv, admin := chainServers(t, 45.8, 46.6, 47.3)
	srv.RouteEst = &routeest.Client{BaseURL: pacedRoads(t).URL}
	srv.Cfg.Set(settings.RouteKey, "on")
	driveMin := func(extra string) float64 {
		_, out := do(t, admin, "POST", ts.URL+"/api/trip-chain", chainBody(extra), xh)
		var r struct {
			Days []struct{ DriveMin float64 }
		}
		if json.Unmarshal([]byte(out), &r) != nil || len(r.Days) == 0 {
			t.Fatalf("plan: %s", out)
		}
		var sum float64
		for _, d := range r.Days {
			sum += d.DriveMin
		}
		return sum
	}
	atLimit, five, none := driveMin(`,"paceKph":0`), driveMin(`,"paceKph":16`), driveMin(``)
	if five >= atLimit || atLimit/five < 1.1 || atLimit/five > 1.2 {
		t.Fatalf("16 km/h on top of 100 km/h should take about 14%% less time: %.1f then %.1f", atLimit, five)
	}
	if none >= atLimit || none <= five {
		t.Fatalf("unset means the default of about 5 mph, between the two: %.1f %.1f %.1f", atLimit, none, five)
	}
}

func TestPaceRaisesTheEnergyOfAHighwayLegFromTheCarsOwnSpeeds(t *testing.T) {
	r := &roadLegs{paceKph: 16, energyAt: func(kph float64) float64 { return kph / 600 }} // 0.1667 kWh/km at 100, 0.1933 at 116
	slow, e0 := r.atPace(routeest.Pace{Km: 70, Minutes: 82.5, HighwayKm: 60, HighwayMin: 36})
	r.paceKph = 0
	same, e1 := r.atPace(routeest.Pace{Km: 70, Minutes: 82.5, HighwayKm: 60, HighwayMin: 36})
	if same != 82.5 || slow >= same {
		t.Errorf("a pace shortens only the highway part: %.2f against %.2f", slow, same)
	}
	if e0 <= e1 || e0 > e1*1.2 {
		t.Errorf("driving faster uses more energy per km: %.4f against %.4f", e0, e1)
	}
}
