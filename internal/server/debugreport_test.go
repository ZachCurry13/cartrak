package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/store"
)

func TestTheDebugReportListsReadingsAndLockFieldsButNeverPositionsOrTheVIN(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	lat, lon, soc := 44.123456, -93.654321, 70.0
	locked, unlocked := true, false
	mk := func(min int, l *bool, ign string) {
		st := &ford.State{TS: time.Now().Add(time.Duration(-min) * time.Minute), SoCPct: &soc, Lat: &lat, Lon: &lon, Locked: l, Ignition: ign,
			Extra: map[string]any{"doorLockStatus[0].value": "UNLOCKED", "doorLockStatus[0].vehicledoor": "UNSPECIFIED_FRONT", "vin": "SECRETVIN123", "position.latitude": 44.123456}, Raw: []byte(`{}`)}
		if _, err := srv.S.SaveState("t", st); err != nil {
			t.Fatal(err)
		}
	}
	mk(30, &locked, "OFF")
	mk(20, &locked, "OFF")
	mk(10, &unlocked, "OFF")
	code, b := do(t, admin, "GET", ts.URL+"/api/debug-report?hours=2", "", nil)
	var out struct {
		Text     string
		Readings int
	}
	if code != 200 || json.Unmarshal([]byte(b), &out) != nil || out.Readings != 3 {
		t.Fatalf("%d %s", code, b)
	}
	for _, want := range []string{"CarTrak debug report", "READINGS", "locked", "UNLOCKED", "LOCKS AND DOORS", "locks: front unlocked", "doorLockStatus[0].value = UNLOCKED", "TRIPS", "CHARGING", "ALERTS SENT", "the same 1 more time"} {
		if !strings.Contains(out.Text, want) {
			t.Errorf("missing %q in:\n%s", want, out.Text)
		}
	}
	for _, never := range []string{"SECRETVIN123", "44.1234", "-93.6543"} {
		if strings.Contains(out.Text, never) {
			t.Errorf("must not contain %q:\n%s", never, out.Text)
		}
	}
	// A viewer cannot read it.
	do(t, admin, "POST", ts.URL+"/api/users", `{"name":"fam","password":"viewer-password-1","role":"viewer"}`, xh)
	vj, _ := cookieJar()
	v := &http.Client{Jar: vj}
	do(t, v, "POST", ts.URL+"/api/login", `{"name":"fam","password":"viewer-password-1"}`, nil)
	if code, _ := do(t, v, "GET", ts.URL+"/api/debug-report", "", nil); code != 403 {
		t.Fatalf("viewer: %d", code)
	}
}

func TestTheDebugReportListsEveryFieldOfTheLatestReadingMinusPositionsAndIdentifiers(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	soc := 70.0
	st := &ford.State{TS: time.Now(), SoCPct: &soc, Raw: []byte(`{}`),
		Extra: map[string]any{"xevBatteryRange.value": 301, "tirePressure[0].value": 245, "vin": "SECRETVIN123", "vehicleLocation.latitude": 44.1, "vehicleLocation.longitude": -93.6}}
	if _, err := srv.S.SaveState("t", st); err != nil {
		t.Fatal(err)
	}
	_, b := do(t, admin, "GET", ts.URL+"/api/debug-report?hours=1", "", nil)
	for _, want := range []string{"EVERY FIELD IN THE LATEST READING (2)", "xevBatteryRange.value = 301", "tirePressure[0].value = 245"} {
		if !strings.Contains(b, want) {
			t.Errorf("missing %q in %s", want, b)
		}
	}
	for _, never := range []string{"SECRETVIN123", "vehicleLocation", "44.1", "-93.6"} {
		if strings.Contains(b, never) {
			t.Errorf("must not contain %q", never)
		}
	}
}

func TestTheDebugReportShowsFastChargerPowerPollingSettingsAndDecisions(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	t0 := time.Now().Add(-2 * time.Hour).Truncate(time.Second)
	fp := func(v float64) *float64 { return &v }
	for i, soc := range []float64{15, 25, 45, 65, 85} {
		st := &ford.State{TS: t0.Add(time.Duration(i*5) * time.Minute), SoCPct: fp(soc), Plug: "CONNECTED", ChargeState: "IN_PROGRESS", ChargeKw: fp(0), EnergyKwh: fp(10 + soc/2), HVVolts: fp(380), Raw: []byte(`{}`),
			Extra: map[string]any{"xevBatteryIoCurrent.value": -300.0, "xevBatteryVoltage.value": 380.0}}
		if _, err := srv.S.SaveState("t", st); err != nil {
			t.Fatal(err)
		}
	}
	c := &store.Charge{StartTS: t0.Unix(), StartSoC: fp(15), StartKwh: fp(17), PeakKw: 114, Station: "DC"}
	if err := srv.S.CreateCharge(c); err != nil {
		t.Fatal(err)
	}
	end := t0.Add(20 * time.Minute).Unix()
	c.EndTS, c.EndSoC, c.EndKwh, c.Open = &end, fp(85), fp(52), false
	if err := srv.S.UpdateCharge(c); err != nil {
		t.Fatal(err)
	}
	srv.S.LogDecision("trip_discarded", "trip 9 thrown away as a blip: 0.00 km in 40 s")
	code, b := do(t, admin, "GET", ts.URL+"/api/debug-report?hours=6", "", nil)
	var out struct{ Text string }
	if code != 200 || json.Unmarshal([]byte(b), &out) != nil {
		t.Fatalf("%d %s", code, b)
	}
	for _, want := range []string{"| ~114.0 |", "kWh left", "power: 20% 114 kW", "charger type DC", "SETTINGS THAT SHAPE BEHAVIOR", "polling: driving every", "POLLING (how often", "DECISIONS CARTRAK MADE", "trip_discarded | trip 9 thrown away"} {
		if !strings.Contains(out.Text, want) {
			t.Fatalf("the report lacks %q:\n%s", want, out.Text)
		}
	}
	if strings.Contains(out.Text, "lat") && strings.Contains(strings.ToLower(out.Text), "latitude =") {
		t.Fatal("no positions")
	}
}
