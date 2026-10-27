package server

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/fuel"
	"github.com/ZachCurry13/cartrak/internal/settings"
)

func TestGasTodayAndTheFillUpComparisonUseTheLiveAveragePrice(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	now := time.Now()
	thisWeek := now.AddDate(0, 0, -2).UTC().Format("2006-01-02")
	lastWeek := now.AddDate(0, 0, -9).UTC().Format("2006-01-02")
	place := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"address":{"city":"Oakdale","ISO3166-2-lvl4":"US-MN"}}`))
	}))
	defer place.Close()
	eia := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"response":{"data":[{"period":%q,"value":"4.000"},{"period":%q,"value":"4.200"}]}}`, lastWeek, thisWeek)
	}))
	defer eia.Close()
	srv.GeoGap, srv.GasURL = time.Millisecond, eia.URL
	srv.Cfg.Set(settings.GeocodeURLKey, place.URL)
	srv.Cfg.Set(settings.GasPriceKey, "5.00")
	srv.Cfg.Set(settings.GasEconomyKey, "25")
	srv.Cfg.Set(settings.EIAKeyKey, "test-key")
	lat, lon, odo := 12.34, 23.45, 1000.0
	if _, err := srv.S.SaveState("t", &ford.State{TS: now.Add(-3 * 24 * time.Hour), OdometerKm: &odo, Lat: &lat, Lon: &lon, Raw: []byte(`{}`)}); err != nil {
		t.Fatal(err)
	}
	paid, gal := 3.90, 10.0
	fill := &fuel.Fill{TS: now.Add(-2 * 24 * time.Hour).Unix(), PricePerGal: &paid, Gallons: &gal, Status: "logged", Source: "manual"}
	if err := srv.S.SaveRefuel(fill); err != nil {
		t.Fatal(err)
	}

	var gn struct {
		Ready     bool     `json:"ready"`
		Area      string   `json:"area"`
		PerGallon float64  `json:"perGallon"`
		Change    *float64 `json:"change"`
		GasPerKm  *float64 `json:"gasPerKm"`
	}
	var fl struct {
		Fills []struct {
			AreaPerGal *float64 `json:"areaPerGal"`
			AreaSource string   `json:"areaSource"`
		} `json:"fills"`
	}
	for i := 0; i < 100; i++ { // the state and the prices are fetched in the background
		_, body := do(t, admin, "GET", ts.URL+"/api/gasnow", "", nil)
		if err := json.Unmarshal([]byte(body), &gn); err != nil {
			t.Fatal(err)
		}
		if gn.Ready {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !gn.Ready || gn.Area != "Minnesota average" || math.Abs(gn.PerGallon-4.2) > 1e-9 || gn.Change == nil || math.Abs(*gn.Change-0.2) > 1e-9 {
		t.Fatalf("gas today: %+v", gn)
	}
	// 25 mpg is 0.09409 L/km; a gallon at 4.20 is 1.1096 a liter
	if gn.GasPerKm == nil || math.Abs(*gn.GasPerKm-0.0940858*4.2/3.785411784) > 1e-4 {
		t.Fatalf("gas cost per km: %v", gn.GasPerKm)
	}
	_, body := do(t, admin, "GET", ts.URL+"/api/fuel", "", nil)
	if err := json.Unmarshal([]byte(body), &fl); err != nil || len(fl.Fills) != 1 || fl.Fills[0].AreaPerGal == nil || math.Abs(*fl.Fills[0].AreaPerGal-4.2) > 1e-9 || fl.Fills[0].AreaSource != "Minnesota average" {
		t.Fatalf("fill-up compared with the area average: %v %s", err, body)
	}
}

func TestGasTodayIsOffWithoutAKey(t *testing.T) {
	ts, _ := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	if _, body := do(t, admin, "GET", ts.URL+"/api/gasnow", "", nil); body != `{"live":false}` && body != "{\"live\":false}\n" {
		t.Fatalf("no key, no live price: %q", body)
	}
}
