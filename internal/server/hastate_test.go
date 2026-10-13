package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
)

func TestTheSmartHomeFeedIsReadOnlyByDeviceTokenAndNeverCarriesPlaceOrVIN(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	_, tok := newDevice(t, ts.URL, admin, "Home Assistant")
	if code, _ := do(t, &http.Client{}, "GET", ts.URL+"/api/ha/state", "", nil); code != 401 {
		t.Fatalf("no token: %d", code)
	}
	_, b := do(t, &http.Client{}, "GET", ts.URL+"/api/ha/state", "", bearer(tok))
	if !strings.Contains(b, `"ready":false`) {
		t.Fatalf("before the first reading: %s", b)
	}
	soc, lat, lon := 64.0, 40.1, -75.2
	srv.S.SaveState("t", &ford.State{TS: time.Now(), SoCPct: &soc, Lat: &lat, Lon: &lon, Plug: "CONNECTED", ChargeState: "CHARGING", Raw: []byte(`{}`)})
	code, b := do(t, &http.Client{}, "GET", ts.URL+"/api/ha/state", "", bearer(tok))
	var out struct {
		Ready      bool
		BatteryPct float64
		Plugged    *bool
		Charging   bool
	}
	if code != 200 || json.Unmarshal([]byte(b), &out) != nil || !out.Ready || out.BatteryPct != 64 || out.Plugged == nil || !*out.Plugged || !out.Charging {
		t.Fatalf("%d %s", code, b)
	}
	if strings.Contains(strings.ToLower(b), "lat") || strings.Contains(strings.ToLower(b), "lon") || strings.Contains(strings.ToLower(b), "vin") {
		t.Fatalf("no location or VIN: %s", b)
	}
	if code, _ := do(t, &http.Client{}, "GET", ts.URL+"/api/ha/state", "", bearer("ct_wrong")); code != 401 {
		t.Fatalf("wrong token: %d", code)
	}
}
