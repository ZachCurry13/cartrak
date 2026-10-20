package server

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMessagesFollowTheUnitsTheDeviceSends(t *testing.T) {
	imperial := httptest.NewRequest("GET", "/", nil)
	metric := httptest.NewRequest("GET", "/", nil)
	metric.Header.Set("X-Units", "metric")
	if metricReq(imperial) || !metricReq(metric) {
		t.Fatal("miles are the default, kilometres only when asked")
	}
	f := routeFailure{RoadKm: 18, OdoKm: 9}
	if got := f.text(false); got != "the road found was 11.2 mi but the odometer says 5.6 mi" {
		t.Fatal(got)
	}
	if got := f.text(true); got != "the road found was 18.0 km but the odometer says 9.0 km" {
		t.Fatal(got)
	}
	if speedText(100, false) != "62 mph" || distText(160.9344, false, 0) != "100 mi" {
		t.Fatal("conversion")
	}
	in := maintIn{EveryKm: new(float64)}
	*in.EveryKm = 5
	if msg := in.valid(false); !strings.Contains(msg, "miles") || strings.Contains(msg, " km") {
		t.Fatal(msg)
	}
}

func TestChargingSettingsHaveTheirOwnListAndPlacesCarryTheirOwnPrice(t *testing.T) {
	ts, cfg := newCfgServer(t, nil)
	c := adminClient(t, ts.URL)
	_, general := do(t, c, "GET", ts.URL+"/api/admin/settings", "", nil)
	_, charging := do(t, c, "GET", ts.URL+"/api/admin/settings?group=charging", "", nil)
	if strings.Contains(general, "CARTRAK_PRICE_HOME") || !strings.Contains(general, "FORD_CLIENT_ID") {
		t.Fatalf("the general list must not carry the charging settings: %s", general)
	}
	if !strings.Contains(charging, "CARTRAK_PRICE_HOME") || strings.Contains(charging, "FORD_CLIENT_ID") {
		t.Fatalf("the charging list holds only charging settings: %s", charging)
	}
	if code, _ := do(t, c, "PUT", ts.URL+"/api/admin/settings?group=charging", `{"values":{"CARTRAK_PRICE_HOME":"0.12"}}`, xh); code != 200 || cfg.PriceHome() == nil || *cfg.PriceHome() != 0.12 {
		t.Fatal("saving a charging setting")
	}
	// a named place with its own price
	if code, body := do(t, c, "POST", ts.URL+"/api/named-places", `{"name":"Cabin","lat":45.1,"lon":-90.2,"chargePrice":0.2}`, xh); code != 200 {
		t.Fatalf("add place: %d %s", code, body)
	}
	_, list := do(t, c, "GET", ts.URL+"/api/named-places", "", nil)
	if !strings.Contains(list, `"chargePrice":0.2`) {
		t.Fatalf("the place keeps its price: %s", list)
	}
	if code, _ := do(t, c, "POST", ts.URL+"/api/named-places", `{"name":"Bad","lat":45.1,"lon":-90.2,"chargePrice":99}`, xh); code != 400 {
		t.Fatal("an absurd price is refused")
	}
}
