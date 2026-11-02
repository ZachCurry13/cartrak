package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/routeest"
	"github.com/ZachCurry13/cartrak/internal/settings"
	"github.com/ZachCurry13/cartrak/internal/weather"
)

// seedUsualDriving records a long drive at 100 km/h at 0.17 kWh/km (5.88 km per kWh) and a battery of 90 kWh, so there is a usual efficiency.
func seedUsualDriving(t *testing.T, srv *Server) {
	t.Helper()
	fp := func(v float64) *float64 { return &v }
	start := time.Now().Add(-5 * time.Hour).Truncate(time.Second)
	odo, kwh := 9000.0, 85.0
	for i := 0; i <= 120; i++ {
		st := &ford.State{TS: start.Add(time.Duration(i*30) * time.Second), OdometerKm: fp(odo), EnergyKwh: fp(kwh), SpeedKph: fp(100), CapacityKwh: fp(90), SoCPct: fp(80),
			Lat: fp(45.0), Lon: fp(-93.2), OutsideC: fp(15), Ignition: "ON", Raw: []byte(`{}`)}
		if _, err := srv.S.SaveState("t", st); err != nil {
			t.Fatal(err)
		}
		d := 100.0 / 3600 * 30
		odo, kwh = odo+d, kwh-d*0.17
	}
}

func TestAMultiStopRouteGetsRoadsForecastRainAndAChargeWhereTheBatteryRunsLow(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	seedUsualDriving(t, srv)
	var legsAsked int
	osrm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		legsAsked++
		w.Write([]byte(`{"code":"Ok","routes":[{"distance":200000,"duration":7200}]}`)) // 200 km, two hours
	}))
	defer osrm.Close()
	wx := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		d := r.URL.Query().Get("start_date")
		var times, a, dir, g, rain, snow, code, sun, temp []string
		for h := 0; h < 24; h++ {
			times = append(times, fmt.Sprintf(`"%sT%02d:00"`, d, h))
			a, dir, g, rain, snow, code, sun, temp = append(a, "10"), append(dir, "90"), append(g, "20"), append(rain, "3"), append(snow, "0"), append(code, "63"), append(sun, "0"), append(temp, "8")
		}
		j := func(v []string) string { return "[" + strings.Join(v, ",") + "]" }
		w.Write([]byte(`{"hourly":{"time":` + j(times) + `,"wind_speed_10m":` + j(a) + `,"wind_direction_10m":` + j(dir) + `,"wind_gusts_10m":` + j(g) + `,"precipitation":` + j(rain) + `,"snowfall":` + j(snow) + `,"weather_code":` + j(code) + `,"shortwave_radiation":` + j(sun) + `,"temperature_2m":` + j(temp) + `}}`))
	}))
	defer wx.Close()
	srv.RouteEst = &routeest.Client{BaseURL: osrm.URL}
	srv.Weather = &weather.Client{ForecastURL: wx.URL, ArchiveURL: wx.URL}
	srv.Cfg.Set(settings.RouteKey, "on")
	srv.Cfg.Set(settings.WeatherKey, "on")

	body := `{"stops":[{"name":"Home","lat":45.0,"lon":-93.2},{"name":"Client","lat":45.0,"lon":-91.0},{"name":"Depot","lat":45.0,"lon":-88.8}],"startPct":60,"reservePct":20,"targetPct":90,"chargeKw":100}`
	code, b := do(t, admin, "POST", ts.URL+"/api/route-plan", body, xh)
	var out struct {
		Legs []struct {
			From, To  string
			Km        float64
			Estimated bool
			ArrivePct float64
			Charge    *struct{ ToPct float64 }
			Weather   struct {
				Known    bool
				Label    string
				TempC    *float64
				TempBand string
				RainPct  float64
			}
		}
		Notes []string
	}
	if code != 200 || json.Unmarshal([]byte(b), &out) != nil || len(out.Legs) != 2 {
		t.Fatalf("%d %s", code, b)
	}
	var raw struct{ Legs []map[string]any }
	if json.Unmarshal([]byte(b), &raw) != nil {
		t.Fatal("not json")
	}
	for _, key := range []string{"from", "to", "km", "minutes", "kwh", "departPct", "arrivePct", "departTs", "arriveTs", "estimated", "weather"} { // the names the page reads, exactly
		if _, ok := raw.Legs[0][key]; !ok {
			t.Fatalf("the leg has no %q: %v", key, raw.Legs[0])
		}
	}
	l0, l1 := out.Legs[0], out.Legs[1]
	if l0.From != "Home" || l0.To != "Client" || l0.Km != 200 || l0.Estimated || legsAsked != 2 {
		t.Fatalf("roads from the routing service: %+v asked %d", l0, legsAsked)
	}
	if !l0.Weather.Known || l0.Weather.Label != "Rain" || l0.Weather.TempC == nil || *l0.Weather.TempC != 8 || l0.Weather.RainPct != 8 || l0.Weather.TempBand == "" {
		t.Fatalf("the forecast for the leg: %+v", l0.Weather)
	}
	// 200 km at 0.17 kWh/km would be 38 kWh of a 90 kWh battery (42%); rain adds 8%, so the leg uses more than that and 60% cannot cover it above 20%.
	if l0.Charge == nil || l0.Charge.ToPct <= 60 {
		t.Fatalf("a charge before the first leg: %+v", l0)
	}
	if l1.ArrivePct < 20 {
		t.Fatalf("the plan keeps the reserve: %+v", l1)
	}
	if len(out.Notes) != 0 {
		t.Fatalf("roads and a forecast were both available: %v", out.Notes)
	}
}

func TestARoutePlanFallsBackToStraightLinesAndSaysSoAndRefusesBadInput(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	seedUsualDriving(t, srv)
	post := func(body string) (int, string) { return do(t, admin, "POST", ts.URL+"/api/route-plan", body, xh) }
	code, b := post(`{"stops":[{"name":"A","lat":45.0,"lon":-93.2},{"name":"B","lat":45.0,"lon":-92.9}],"startPct":80}`)
	var out struct {
		Legs []struct {
			Km        float64
			Estimated bool
		}
		Notes []string
	}
	if code != 200 || json.Unmarshal([]byte(b), &out) != nil || len(out.Legs) != 1 || !out.Legs[0].Estimated || len(out.Notes) != 2 {
		t.Fatalf("%d %s", code, b)
	}
	if out.Legs[0].Km < 30 || out.Legs[0].Km > 32 {
		t.Fatalf("0.3 degrees of longitude at 45 north is about 23.6 km, made 30%% longer: %v", out.Legs[0].Km)
	}
	var many []string
	for i := 0; i < 12; i++ {
		many = append(many, fmt.Sprintf(`{"lat":45.%d,"lon":-93.1}`, i))
	}
	for name, body := range map[string]string{
		"one stop":    `{"stops":[{"lat":45,"lon":-93}],"startPct":80}`,
		"too many":    `{"stops":[` + strings.Join(many, ",") + `,{"lat":46,"lon":-93}],"startPct":80}`,
		"off the map": `{"stops":[{"lat":45,"lon":-93},{"lat":95,"lon":-93}],"startPct":80}`,
		"no charge":   `{"stops":[{"lat":45,"lon":-93},{"lat":45.1,"lon":-93}]}`,
		"null island": `{"stops":[{"lat":45,"lon":-93},{"lat":0,"lon":0}],"startPct":80}`,
	} {
		if c, _ := post(body); c != 400 {
			t.Errorf("%s was accepted (%d)", name, c)
		}
	}
}
