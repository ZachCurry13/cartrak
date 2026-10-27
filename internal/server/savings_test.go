package server

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/settings"
	"github.com/ZachCurry13/cartrak/internal/store"
)

func TestSavingsViewWorksOutCostPerDistanceAndTheSavingAgainstAGasCar(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	loc := srv.defaultLoc()
	now := time.Now().In(loc)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
	if now.Sub(monthStart) < 6*time.Hour {
		t.Skip("the first hours of a month leave no room for this month's sessions")
	}
	srv.Cfg.Set(settings.PriceHomeKey, "0.15")
	srv.Cfg.Set(settings.GasPriceKey, "3.50")
	srv.Cfg.Set(settings.GasEconomyKey, "25")
	for i, odo := range []float64{1000, 1100} {
		if _, err := srv.S.SaveState("t", &ford.State{TS: monthStart.Add(time.Duration(i+1) * time.Hour), OdometerKm: &odo, Raw: []byte(`{}`)}); err != nil {
			t.Fatal(err)
		}
	}
	// a drive that used 25 kWh over 100 km: the car goes 4 km on each kWh
	tripEnd, e0, e1 := monthStart.Add(40*time.Minute).Unix(), 50.0, 25.0
	tr := &store.Trip{StartTS: monthStart.Add(10 * time.Minute).Unix(), StartKwh: &e0, Category: "personal", CategorySource: "manual"}
	if err := srv.S.CreateTrip(tr); err != nil {
		t.Fatal(err)
	}
	tr.EndTS, tr.DistanceKm, tr.Open, tr.EndKwh = &tripEnd, 100, false, &e1
	if err := srv.S.UpdateTrip(tr); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		from, to := 40.0, 60.0
		c := &store.Charge{StartTS: monthStart.Add(time.Duration(3+i) * time.Hour).Unix(), StartKwh: &from, PeakKw: 7}
		if err := srv.S.CreateCharge(c); err != nil {
			t.Fatal(err)
		}
		end := c.StartTS + 1800
		c.EndTS, c.EndKwh, c.Open = &end, &to, false
		if err := srv.S.UpdateCharge(c); err != nil {
			t.Fatal(err)
		}
	}
	code, body := do(t, admin, "GET", ts.URL+"/api/savings", "", nil)
	var v struct {
		GasSet bool `json:"gasSet"`
		Month  struct {
			Km        float64  `json:"km"`
			CostPerKm *float64 `json:"costPerKm"`
			Saved     *float64 `json:"saved"`
			CO2       *float64 `json:"co2AvoidedKg"`
		} `json:"month"`
	}
	if code != 200 || json.Unmarshal([]byte(body), &v) != nil {
		t.Fatalf("%d %s", code, body)
	}
	// 2 sessions of 20 kWh into the battery at 90% = 22.2 kWh each from the wall at 0.15 = 3.33 each, 6.67 in all, over 100 km
	if !v.GasSet || math.Abs(v.Month.Km-100) > 0.01 || v.Month.CostPerKm == nil || math.Abs(*v.Month.CostPerKm-0.0667) > 0.001 {
		t.Fatalf("cost per km: %s", body)
	}
	// the two charges added 40 kWh, which at 4 km per kWh is 160 km: 99.4 miles at 25 mpg is 3.977 gallons, 13.92 at 3.50;
	// saved = 13.92 - 6.67, whatever the distance driven was
	if v.Month.Saved == nil || math.Abs(*v.Month.Saved-7.25) > 0.05 || v.Month.CO2 == nil || *v.Month.CO2 <= 0 {
		t.Fatalf("saving: %s", body)
	}
}

func TestSavingsViewDoesNotCompareWithoutAGasCar(t *testing.T) {
	ts, _ := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	_, body := do(t, admin, "GET", ts.URL+"/api/savings", "", nil)
	var v struct {
		GasSet bool `json:"gasSet"`
		Year   struct {
			Saved *float64 `json:"saved"`
		} `json:"year"`
	}
	if err := json.Unmarshal([]byte(body), &v); err != nil || v.GasSet || v.Year.Saved != nil {
		t.Fatalf("nothing to compare with: %s", body)
	}
}

func TestPlanAheadNeedsEnoughHistoryAndThenForecastsTheWeek(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	soc, capKwh := 60.0, 80.0
	first := time.Now().AddDate(0, 0, -25)
	if _, err := srv.S.SaveState("t", &ford.State{TS: first, SoCPct: &soc, CapacityKwh: &capKwh, Raw: []byte(`{}`)}); err != nil {
		t.Fatal(err)
	}
	if _, err := srv.S.SaveState("t", &ford.State{TS: time.Now(), SoCPct: &soc, CapacityKwh: &capKwh, Raw: []byte(`{}`)}); err != nil {
		t.Fatal(err)
	}
	type reply struct {
		Ready    bool `json:"ready"`
		Forecast struct {
			Days     []struct{ Km float64 } `json:"days"`
			TotalKwh float64                `json:"totalKwh"`
		} `json:"forecast"`
	}
	get := func() reply {
		_, b := do(t, admin, "GET", ts.URL+"/api/plan-ahead", "", nil)
		var r reply
		if err := json.Unmarshal([]byte(b), &r); err != nil {
			t.Fatalf("%s", b)
		}
		return r
	}
	if get().Ready {
		t.Fatal("no trips, so nothing to forecast from")
	}
	kwh := func(x float64) *float64 { return &x }
	for d := 1; d <= 25; d++ { // a 20 km trip using 5 kWh every day since the first reading
		n := time.Now().In(srv.defaultLoc())
		start := time.Date(n.Year(), n.Month(), n.Day(), 12, 0, 0, 0, n.Location()).AddDate(0, 0, -d) // midday, so the day is never in doubt
		finishedTrip(t, srv, start, 20, func(tr *store.Trip) { tr.StartKwh, tr.EndKwh = kwh(50), kwh(45) })
	}
	r := get()
	if !r.Ready || len(r.Forecast.Days) != 7 {
		t.Fatalf("a week: %+v", r)
	}
	// 20 km a day at 0.25 kWh per km: 5 kWh a day, 35 for the week
	if math.Abs(r.Forecast.TotalKwh-35) > 0.5 || math.Abs(r.Forecast.Days[0].Km-20) > 0.01 {
		t.Fatalf("forecast: %+v", r.Forecast)
	}
}
