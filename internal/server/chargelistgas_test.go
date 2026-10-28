package server

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/settings"
	"github.com/ZachCurry13/cartrak/internal/store"
)

func TestEachSessionInTheChargingListCarriesItsGasComparison(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	loc := srv.defaultLoc()
	day := time.Now().In(loc).AddDate(0, 0, -3)
	start := time.Date(day.Year(), day.Month(), day.Day(), 9, 0, 0, 0, loc)
	srv.Cfg.Set(settings.GasPriceKey, "4.00")
	srv.Cfg.Set(settings.GasEconomyKey, "25")
	srv.Cfg.Set(settings.PriceHomeKey, "0.15")

	e0, e1 := 50.0, 25.0
	tr := &store.Trip{StartTS: start.Unix(), StartKwh: &e0, Category: "personal", CategorySource: "manual"}
	if err := srv.S.CreateTrip(tr); err != nil {
		t.Fatal(err)
	}
	end := start.Add(30 * time.Minute).Unix()
	tr.EndTS, tr.DistanceKm, tr.Open, tr.EndKwh = &end, 100, false, &e1 // 4 km per kWh
	if err := srv.S.UpdateTrip(tr); err != nil {
		t.Fatal(err)
	}
	from, to := 40.0, 60.0 // 20 kWh added: 80 km
	c := &store.Charge{StartTS: start.Add(3 * time.Hour).Unix(), StartKwh: &from, PeakKw: 7}
	if err := srv.S.CreateCharge(c); err != nil {
		t.Fatal(err)
	}
	cEnd := c.StartTS + 1800
	c.EndTS, c.EndKwh, c.Open = &cEnd, &to, false
	if err := srv.S.UpdateCharge(c); err != nil {
		t.Fatal(err)
	}

	_, body := do(t, admin, "GET", ts.URL+"/api/charges", "", nil)
	var list []struct {
		Gas *struct {
			Cost float64 `json:"cost"`
			Km   float64 `json:"km"`
		} `json:"gas"`
	}
	if err := json.Unmarshal([]byte(body), &list); err != nil || len(list) != 1 || list[0].Gas == nil {
		t.Fatalf("the session carries a gas comparison (%v): %s", err, body)
	}
	if want := 1.988 * 4.00; math.Abs(list[0].Gas.Cost-want) > 0.1 || list[0].Gas.Km < 79 || list[0].Gas.Km > 81 {
		t.Fatalf("gas %.2f for %.1f km, want about %.2f for 80", list[0].Gas.Cost, list[0].Gas.Km, want)
	}
}
