package server

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
)

func TestTheFieldWatcherListsWhatTheCarSendsAndWhatChanged(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	t0 := time.Now().Add(-2 * time.Hour)
	for i, soc := range []float64{80, 80, 79} {
		st := &ford.State{TS: t0.Add(time.Duration(i*10) * time.Minute), SoCPct: &soc, Raw: []byte(`{}`),
			Extra: map[string]any{"telemetry.metrics.xevBatteryStateOfCharge.value": soc, "telemetry.metrics.xevBatteryStateOfCharge.updateTime": "2026-10-19T10:00:00Z",
				"telemetry.metrics.oilLifeRemaining.value": 100.0}}
		if _, err := srv.S.SaveState("t", st); err != nil {
			t.Fatal(err)
		}
	}
	code, b := do(t, admin, "GET", ts.URL+"/api/fields?hours=6", "", nil)
	var out struct {
		Readings int
		Fields   []struct {
			Metric  string
			Changes int
			UsedFor string
			Samples int
		}
	}
	if code != 200 || json.Unmarshal([]byte(b), &out) != nil || out.Readings != 3 {
		t.Fatalf("%d %s", code, b)
	}
	got := map[string]struct {
		Changes int
		UsedFor string
		Samples int
	}{}
	for _, f := range out.Fields {
		got[f.Metric] = struct {
			Changes int
			UsedFor string
			Samples int
		}{f.Changes, f.UsedFor, f.Samples}
	}
	if g := got["xevBatteryStateOfCharge"]; g.Changes != 1 || g.UsedFor == "" || g.Samples != 3 {
		t.Fatalf("a used field that changed once: %+v", got)
	}
	if g, ok := got["oilLifeRemaining"]; !ok || g.Changes != 0 || g.UsedFor != "" {
		t.Fatalf("a field nothing uses and that never changed is still listed: %+v", got)
	}
}
