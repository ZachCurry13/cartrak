package server

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/settings"
	"github.com/ZachCurry13/cartrak/internal/store"
)

func TestADayShowsWhatItsChargingSavedAgainstTheGasCar(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	loc := srv.defaultLoc()
	day := time.Now().In(loc).AddDate(0, 0, -2)
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
	from, to := 40.0, 60.0 // 20 kWh added
	c := &store.Charge{StartTS: start.Add(3 * time.Hour).Unix(), StartKwh: &from, PeakKw: 7}
	if err := srv.S.CreateCharge(c); err != nil {
		t.Fatal(err)
	}
	cEnd := c.StartTS + 1800
	c.EndTS, c.EndKwh, c.Open = &cEnd, &to, false
	if err := srv.S.UpdateCharge(c); err != nil {
		t.Fatal(err)
	}

	_, body := do(t, admin, "GET", ts.URL+"/api/day?date="+start.Format("2006-01-02"), "", nil)
	var v struct {
		Events []struct {
			Kind string `json:"kind"`
			Gas  *struct {
				Cost float64 `json:"cost"`
				Km   float64 `json:"km"`
			} `json:"gas"`
		} `json:"events"`
		Savings *struct {
			GasCost *float64 `json:"gasCost"`
			Saved   *float64 `json:"saved"`
			EquivKm *float64 `json:"equivKm"`
		} `json:"savings"`
	}
	if err := json.Unmarshal([]byte(body), &v); err != nil || v.Savings == nil || v.Savings.GasCost == nil || v.Savings.Saved == nil {
		t.Fatalf("the day carries the comparison (%v): %s", err, body)
	}
	// 20 kWh at 4 km/kWh is 80 km: 49.7 miles at 25 mpg is 1.988 gallons at 4.00
	if want := 1.988 * 4.00; math.Abs(*v.Savings.GasCost-want) > 0.1 || *v.Savings.EquivKm < 79 || *v.Savings.EquivKm > 81 {
		t.Fatalf("gas cost %.2f (want about %.2f), range %.1f km", *v.Savings.GasCost, want, *v.Savings.EquivKm)
	}
	withGas := 0
	for _, e := range v.Events {
		if e.Kind == "charging" && e.Gas != nil && math.Abs(e.Gas.Cost-1.988*4.00) < 0.1 && e.Gas.Km > 79 && e.Gas.Km < 81 {
			withGas++
		}
	}
	if withGas != 1 {
		t.Fatalf("the charge itself carries the comparison: %s", body)
	}
	// a day with no charging has none
	_, body = do(t, admin, "GET", ts.URL+"/api/day?date="+start.AddDate(0, 0, -1).Format("2006-01-02"), "", nil)
	if err := json.Unmarshal([]byte(body), &v); err != nil || (v.Savings != nil && v.Savings.GasCost != nil) {
		t.Fatalf("no charging, no comparison: %s", body)
	}
}
