package server

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/settings"
	"github.com/ZachCurry13/cartrak/internal/store"
)

func TestLiveGasPriceFollowsWhereTheChargeHappenedAndTheOwnersAdjustment(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	loc := srv.defaultLoc()
	now := time.Now().In(loc)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
	if now.Sub(monthStart) < 6*time.Hour {
		t.Skip("the first hours of a month leave no room for this month's sessions")
	}
	week := monthStart.AddDate(0, 0, -7).UTC().Format("2006-01-02")

	place := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"address":{"city":"Springfield","ISO3166-2-lvl4":"US-MI"}}`))
	}))
	defer place.Close()
	var eiaKey, eiaArea string
	eia := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		eiaKey, eiaArea = r.URL.Query().Get("api_key"), r.URL.Query().Get("facets[duoarea][]")
		fmt.Fprintf(w, `{"response":{"data":[{"period":%q,"value":"3.7854"}]}}`, week)
	}))
	defer eia.Close()
	srv.GeoGap, srv.GasURL = time.Millisecond, eia.URL
	srv.Cfg.Set(settings.GeocodeURLKey, place.URL)
	srv.Cfg.Set(settings.GasPriceKey, "5.00") // the owner's own price, used until the live one is known
	srv.Cfg.Set(settings.GasEconomyKey, "25")
	srv.Cfg.Set(settings.PriceHomeKey, "0.15")
	srv.Cfg.Set(settings.EIAKeyKey, "test-key")
	srv.Cfg.Set(settings.GasAdjustKey, "Springfield +10%")

	e0, e1 := 50.0, 25.0
	tr := &store.Trip{StartTS: monthStart.Add(10 * time.Minute).Unix(), StartKwh: &e0, Category: "personal", CategorySource: "manual"}
	if err := srv.S.CreateTrip(tr); err != nil {
		t.Fatal(err)
	}
	tripEnd := monthStart.Add(40 * time.Minute).Unix()
	tr.EndTS, tr.DistanceKm, tr.Open, tr.EndKwh = &tripEnd, 100, false, &e1
	if err := srv.S.UpdateTrip(tr); err != nil {
		t.Fatal(err)
	}
	lat, lon, from, to := 12.34, 23.45, 40.0, 60.0
	c := &store.Charge{StartTS: monthStart.Add(3 * time.Hour).Unix(), StartKwh: &from, PeakKw: 7, Lat: &lat, Lon: &lon}
	if err := srv.S.CreateCharge(c); err != nil {
		t.Fatal(err)
	}
	end := c.StartTS + 1800
	c.EndTS, c.EndKwh, c.Open = &end, &to, false
	if err := srv.S.UpdateCharge(c); err != nil {
		t.Fatal(err)
	}

	var v struct {
		Live  bool `json:"live"`
		Month struct {
			GasCost *float64 `json:"gasCost"`
			Sources []string `json:"sources"`
		} `json:"month"`
	}
	// The state and the weekly prices are fetched in the background; the page uses the owner's price until they are known.
	for i := 0; i < 100; i++ {
		_, body := do(t, admin, "GET", ts.URL+"/api/savings", "", nil)
		v.Month.Sources = nil
		if err := json.Unmarshal([]byte(body), &v); err != nil {
			t.Fatal(err)
		}
		if len(v.Month.Sources) == 1 && v.Month.Sources[0] != "your gas price" {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !v.Live || len(v.Month.Sources) != 1 || v.Month.Sources[0] != "Midwest average +10%" || eiaKey != "test-key" || eiaArea != "R20" {
		t.Fatalf("a Michigan charge uses the Midwest average with the owner's +10%%: %+v key %q area %q", v, eiaKey, eiaArea)
	}
	// 20 kWh at 4 km/kWh is 80 km: 49.7 miles at 25 mpg is 1.988 gallons, at 3.7854 x 1.10 a gallon
	if v.Month.GasCost == nil || math.Abs(*v.Month.GasCost-1.988*3.7854*1.10) > 0.05 {
		t.Fatalf("gas cost: %v", v.Month.GasCost)
	}
}
