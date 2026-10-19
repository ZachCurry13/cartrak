package server

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
)

func TestBatteryTruthComparesTheDisplayWithTheBatterysOwnLevel(t *testing.T) {
	truthMu.Lock()
	truthCache, truthAt = nil, time.Time{}
	truthMu.Unlock()
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	soc := 67.5
	st := &ford.State{TS: time.Now().Add(-time.Hour), SoCPct: &soc, Raw: []byte(`{}`), Extra: map[string]any{
		"telemetry.metrics.xevBatteryStateOfCharge.value":       67.5,
		"telemetry.metrics.xevBatteryActualStateOfCharge.value": 64.97,
		"telemetry.metrics.xevBatteryEnergyRemaining.value":     55.45,
		"telemetry.metrics.xevBatteryCapacity.value":            88.75,
		"telemetry.metrics.xevBatteryMaximumRange.value":        393.8}}
	if _, err := srv.S.SaveState("t", st); err != nil {
		t.Fatal(err)
	}
	code, b := do(t, admin, "GET", ts.URL+"/api/battery-truth", "", nil)
	var out struct {
		Now struct {
			DisplayedSoC, ActualSoC, UsableKwh, FordKwh *float64
		}
	}
	if code != 200 || json.Unmarshal([]byte(b), &out) != nil {
		t.Fatalf("%d %s", code, b)
	}
	if out.Now.DisplayedSoC == nil || *out.Now.DisplayedSoC != 67.5 || out.Now.ActualSoC == nil || *out.Now.ActualSoC != 64.97 {
		t.Fatalf("%s", b)
	}
	if out.Now.UsableKwh == nil || math.Abs(*out.Now.UsableKwh-55.45/0.6497) > 0.01 {
		t.Fatalf("energy left over the true level: %v", out.Now.UsableKwh)
	}
	// Near the ends the level is too coarse to divide by.
	if _, ok := usable(map[string]any{"xevBatteryActualStateOfCharge.value": 99.0, "xevBatteryEnergyRemaining.value": 87.0}); ok {
		t.Fatal("99 percent gives nothing")
	}
}
