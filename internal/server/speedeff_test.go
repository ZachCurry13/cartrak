package server

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
)

func TestSpeedEfficiencyComparesBandsFromStoredDrives(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	fp := func(v float64) *float64 { return &v }
	t0 := time.Now().Add(-2 * time.Hour).Truncate(time.Second)
	odo, kwh := 5000.0, 80.0
	for i := 0; i <= 200; i++ { // 100 minutes at 112 km/h using 0.22 kWh per km
		st := &ford.State{TS: t0.Add(time.Duration(i*30) * time.Second), OdometerKm: fp(odo), EnergyKwh: fp(kwh), SpeedKph: fp(112), Ignition: "ON", Raw: []byte(`{}`)}
		if _, err := srv.S.SaveState("t", st); err != nil {
			t.Fatal(err)
		}
		d := 112.0 / 3600 * 30
		odo += d
		kwh -= d * 0.22
	}
	code, b := do(t, admin, "GET", ts.URL+"/api/speed-efficiency?days=30", "", nil)
	var out struct {
		Bands []struct{ LoKph, KmPerKwh float64 }
	}
	if code != 200 || json.Unmarshal([]byte(b), &out) != nil || len(out.Bands) != 1 {
		t.Fatalf("%d %s", code, b)
	}
	if out.Bands[0].LoKph != 110 || out.Bands[0].KmPerKwh < 4.4 || out.Bands[0].KmPerKwh > 4.7 {
		t.Fatalf("112 km/h at 0.22 kWh per km is about 4.5 km per kWh: %+v", out.Bands[0])
	}
}
