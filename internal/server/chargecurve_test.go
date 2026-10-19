package server

import (
	"encoding/json"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/store"
)

func TestTheChargingCurveReadsPowerFromThePackWhenTheChargerReportsNone(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	t0 := time.Now().Add(-3 * time.Hour).Truncate(time.Minute)
	fp := func(v float64) *float64 { return &v }
	kwhLeft := 10.0
	for i := 0; i <= 6; i++ {
		soc := 10 + float64(i)*12
		st := &ford.State{TS: t0.Add(time.Duration(i*5) * time.Minute), SoCPct: fp(soc), Plug: "CONNECTED", ChargeState: "IN_PROGRESS", ChargeKw: fp(0), HVVolts: fp(370),
			EnergyKwh: fp(kwhLeft + float64(i)*10), Raw: []byte(`{}`),
			Extra: map[string]any{"xevBatteryIoCurrent.value": -(120.0 - float64(i)*15), "xevBatteryVoltage.value": 370.0}}
		if _, err := srv.S.SaveState("t", st); err != nil {
			t.Fatal(err)
		}
	}
	end := t0.Add(30 * time.Minute).Unix()
	c := &store.Charge{StartTS: t0.Unix(), StartKwh: fp(10), StartSoC: fp(10), PeakKw: 0}
	if err := srv.S.CreateCharge(c); err != nil {
		t.Fatal(err)
	}
	c.EndTS, c.EndKwh, c.EndSoC, c.Open = &end, fp(70), fp(82), false
	if err := srv.S.UpdateCharge(c); err != nil {
		t.Fatal(err)
	}
	code, b := do(t, admin, "GET", fmt.Sprintf("%s/api/charges/%d/curve", ts.URL, c.ID), "", nil)
	var out struct {
		Points []struct{ SoC, Kw float64 }
		PeakKw float64
		AvgKw  *float64
	}
	if code != 200 || json.Unmarshal([]byte(b), &out) != nil || len(out.Points) != 7 {
		t.Fatalf("%d %s", code, b)
	}
	if math.Abs(out.PeakKw-44.4) > 0.1 { // 120 A at 370 V
		t.Fatalf("peak from the pack current: %v", out.PeakKw)
	}
	if out.AvgKw == nil || math.Abs(*out.AvgKw-120) > 1 { // 60 kWh in half an hour
		t.Fatalf("average from the energy: %v", out.AvgKw)
	}
	if code, _ := do(t, admin, "GET", ts.URL+"/api/charges/999/curve", "", nil); code != 404 {
		t.Fatalf("no such session: %d", code)
	}
	code, b = do(t, admin, "GET", ts.URL+"/api/charging/fast-average", "", nil)
	if code != 200 {
		t.Fatalf("%d %s", code, b)
	}
}
