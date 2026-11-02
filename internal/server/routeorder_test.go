package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/ZachCurry13/cartrak/internal/routeest"
	"github.com/ZachCurry13/cartrak/internal/settings"
)

// A road service for places along one line of latitude: one degree of longitude is an hour and 80 km.
func lineRoads(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		last := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
		var lons []float64
		for _, p := range strings.Split(last, ";") {
			v, _ := strconv.ParseFloat(strings.Split(p, ",")[0], 64)
			lons = append(lons, v)
		}
		abs := func(x float64) float64 {
			if x < 0 {
				return -x
			}
			return x
		}
		if strings.Contains(r.URL.Path, "/table/") {
			var dur, dis []string
			for _, a := range lons {
				var d, k []string
				for _, b := range lons {
					d, k = append(d, fmt.Sprint(abs(a-b)*3600)), append(k, fmt.Sprint(abs(a-b)*80000))
				}
				dur, dis = append(dur, "["+strings.Join(d, ",")+"]"), append(dis, "["+strings.Join(k, ",")+"]")
			}
			w.Write([]byte(`{"code":"Ok","durations":[` + strings.Join(dur, ",") + `],"distances":[` + strings.Join(dis, ",") + `]}`))
			return
		}
		w.Write([]byte(fmt.Sprintf(`{"code":"Ok","routes":[{"distance":%v,"duration":%v}]}`, abs(lons[0]-lons[1])*80000, abs(lons[0]-lons[1])*3600)))
	}))
}

func TestTheBestOrderRoundTripReordersTheCustomersAndComesBackHome(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	seedUsualDriving(t, srv)
	roads := lineRoads(t)
	defer roads.Close()
	srv.RouteEst = &routeest.Client{BaseURL: roads.URL}
	srv.Cfg.Set(settings.RouteKey, "on")

	// Home at -93.2, then customers given in a zigzag: -91.0, -92.0, -90.0. Any best round trip is 6.4 hours (out to -90.0 and back).
	body := `{"stops":[{"name":"Home","lat":45,"lon":-93.2},{"name":"Ames","lat":45,"lon":-91.0},{"name":"Bell","lat":45,"lon":-92.0},{"name":"Cole","lat":45,"lon":-90.0}],"startPct":90,"optimize":true,"roundTrip":true}`
	code, b := do(t, admin, "POST", ts.URL+"/api/route-plan", body, xh)
	var out struct {
		Stops        []struct{ Name string }
		Order        []int
		SavedMinutes float64
		RoundTrip    bool
		Legs         []struct{ From, To string }
		TotalKm      float64
	}
	if code != 200 || json.Unmarshal([]byte(b), &out) != nil {
		t.Fatalf("%d %s", code, b)
	}
	names := []string{}
	for _, s := range out.Stops {
		names = append(names, s.Name)
	}
	seen := map[string]bool{}
	for _, n := range names[1 : len(names)-1] {
		seen[n] = true
	}
	if names[0] != "Home" || names[len(names)-1] != "Home" || len(names) != 5 || len(seen) != 3 || !out.RoundTrip || len(out.Legs) != 4 || fmt.Sprint(out.Order) == "[0 1 2 3]" {
		t.Fatalf("every customer once, reordered, and back home: %v %v legs %d", names, out.Order, len(out.Legs))
	}
	// Given order: 1.8 + 1 + ... home->Ames 2.2h, Ames->Bell 1h, Bell->Cole 2h, Cole->home 3.2h = 8.4h. Best: out 3.2h and back 3.2h = 6.4h, saving 2 hours.
	if out.SavedMinutes < 119 || out.SavedMinutes > 121 {
		t.Fatalf("it saves two hours of driving over the order given: %v", out.SavedMinutes)
	}
	if out.Legs[3].To != "Home" {
		t.Fatalf("the last leg is the way home: %+v", out.Legs[3])
	}

	// Without asking for the best order the customers stay as given, and the way home is still added.
	code, b = do(t, admin, "POST", ts.URL+"/api/route-plan", strings.Replace(body, `"optimize":true`, `"optimize":false`, 1), xh)
	out.Stops = nil
	json.Unmarshal([]byte(b), &out)
	names = nil
	for _, s := range out.Stops {
		names = append(names, s.Name)
	}
	if code != 200 || strings.Join(names, ",") != "Home,Ames,Bell,Cole,Home" || out.SavedMinutes != 0 {
		t.Fatalf("%d %v", code, names)
	}
}
