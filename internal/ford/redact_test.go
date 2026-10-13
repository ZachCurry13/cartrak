package ford

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestProbeRedactsLocationAndVIN(t *testing.T) {
	f := newFake2(t)
	c := f.client(memKV{refreshKey: "R0"})
	res := c.Probe(context.Background())
	b, _ := json.Marshal(res)
	s := string(b)
	for _, leak := range []string{"1FMTK1234ABCD5678", "42.3", "-83.2"} {
		if strings.Contains(s, leak) {
			t.Fatalf("probe leaked %q: %s", leak, s)
		}
	}
	if !strings.Contains(s, "81.5") || !strings.Contains(s, "XEV_BATTERY_STATE_OF_CHARGE") {
		t.Fatalf("probe should keep useful values: %s", s)
	}
	if res["telemetry"].Status != 200 || len(res) != 6 {
		t.Fatalf("probe coverage: %d endpoints, telemetry=%d", len(res), res["telemetry"].Status)
	}
}

func TestRedactTokens(t *testing.T) {
	got := tokens("vehicleLongitude_valueGPS")
	want := []string{"vehicle", "longitude", "value", "gps"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("tokens: %v", got)
	}
	if !sensitive("latitude") || !sensitive("positionValue") || sensitive("chargePlatform") || sensitive("batteryTemperature") {
		t.Fatal("sensitivity rules wrong")
	}
}
