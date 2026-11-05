package server

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
)

func TestTheRestingVoltageCountsOnlyReadingsAfterTheCarHasBeenIdle(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	fp := func(v float64) *float64 { return &v }
	start := time.Now().Add(-5 * time.Hour).Truncate(time.Second)
	save := func(min int, ig string, speed, volts float64) {
		st := &ford.State{TS: start.Add(time.Duration(min) * time.Minute), OdometerKm: fp(5000), SpeedKph: fp(speed), Ignition: ig, Battery12V: fp(volts), SoCPct: fp(70), Raw: []byte(`{}`)}
		if _, err := srv.S.SaveState("t", st); err != nil {
			t.Fatal(err)
		}
	}
	save(0, "Run", 100, 14.2)  // driving: the charging system, not the battery
	save(10, "Off", 0, 13.9)   // just parked: ten minutes is not idle
	save(45, "Off", 0, 12.5)   // idle for 35 minutes: counts
	save(90, "Off", 0, 12.3)   // counts
	save(120, "Run", 60, 14.4) // driving again
	save(130, "Off", 0, 13.8)  // not idle
	_, b := do(t, admin, "GET", ts.URL+"/api/vehicle-health", "", nil)
	var r struct {
		Resting *struct {
			Avg, Min, Max float64
			N             int
		}
	}
	if json.Unmarshal([]byte(b), &r) != nil || r.Resting == nil {
		t.Fatalf("a resting figure is expected: %s", b)
	}
	if r.Resting.N != 2 || r.Resting.Avg < 12.39 || r.Resting.Avg > 12.41 || r.Resting.Min != 12.3 || r.Resting.Max != 12.5 {
		t.Fatalf("only the two readings after 30 idle minutes count, averaging 12.4: %+v", r.Resting)
	}
}
