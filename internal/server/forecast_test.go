package server

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/settings"
)

func TestChargeForecastPricesTheTopUpAtTheOwnersPrices(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	soc, cap := 30.0, 90.0
	if _, err := srv.S.SaveState("t", &ford.State{TS: time.Now(), SoCPct: &soc, CapacityKwh: &cap, Raw: []byte(`{}`)}); err != nil {
		t.Fatal(err)
	}
	type fc struct {
		OK       bool `json:"ok"`
		Forecast struct {
			BatteryKwh float64  `json:"batteryKwh"`
			HomeCost   *float64 `json:"homeCost"`
		} `json:"forecast"`
		SocNow *float64 `json:"socNow"`
	}
	get := func(q string) fc {
		code, b := do(t, admin, "GET", ts.URL+"/api/charge-forecast"+q, "", nil)
		var out fc
		if code != 200 || json.Unmarshal([]byte(b), &out) != nil {
			t.Fatalf("%d %s", code, b)
		}
		return out
	}
	// Defaults: from the level now (30) to 80 percent. 50 percent of 90 kWh is 45 kWh; no price yet, so no cost.
	f := get("")
	if !f.OK || math.Abs(f.Forecast.BatteryKwh-45) > 1e-6 || f.Forecast.HomeCost != nil || f.SocNow == nil {
		t.Fatalf("%+v", f)
	}
	srv.Cfg.Set(settings.PriceHomeKey, "0.10")
	f = get("?from=20&to=100")
	if !f.OK || f.Forecast.HomeCost == nil || *f.Forecast.HomeCost < 7.99 {
		t.Fatalf("72 kWh at 0.10 and some loss costs over 7.20: %+v", f)
	}
	if get("?from=80&to=20").OK || get("?from=abc").OK {
		t.Fatal("a backwards or unreadable range is not a forecast")
	}
}
