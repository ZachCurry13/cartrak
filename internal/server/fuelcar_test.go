package server

import (
	"strings"
	"testing"

	"github.com/ZachCurry13/cartrak/internal/ford"
)

func TestAGasCarIsOfferedNoElectricReportColumnsAndNoChargingAlertWording(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	_, ev := do(t, admin, "GET", ts.URL+"/api/reports/options", "", nil)
	if !strings.Contains(ev, "Energy Used (kWh)") {
		t.Fatalf("an electric car keeps the energy columns: %.200s", ev)
	}
	_ = srv.S.SetKV(ford.EngineKV, "ICE")
	if !srv.fuelCar() {
		t.Fatal("a gas engine is a fuel car")
	}
	_, gas := do(t, admin, "GET", ts.URL+"/api/reports/options", "", nil)
	if strings.Contains(gas, "kWh") || strings.Contains(gas, "Battery Start") || !strings.Contains(gas, "Odometer Start") {
		t.Fatalf("no energy columns for a gas car, the others stay: %.300s", gas)
	}
	if it := alertsItem(0, true); strings.Contains(it.Detail, "charging") || !strings.Contains(it.Detail, "fuel is low") {
		t.Fatalf("fuel wording: %+v", it)
	}
}
