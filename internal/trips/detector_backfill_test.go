package trips

import (
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/store"
)

func energyReading(ts time.Time, state, plug string, soc, kwh, kw float64, extra map[string]any) *ford.State {
	st := chargeReading(ts, state, plug, soc)
	st.EnergyKwh, st.ChargeKw, st.Extra = &kwh, &kw, extra
	return st
}

func TestAChargeThatBeganBetweenTwoPollsStartsAtTheLastReadingBeforeIt(t *testing.T) {
	s, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	d := &Detector{S: s, TZ: time.UTC}
	t0 := time.Date(2026, 10, 20, 17, 6, 26, 0, time.UTC)
	steps := []*ford.State{
		energyReading(t0, "NOT_READY", "DISCONNECTED", 36, 30.2, 0, nil),
		// ten minutes later the car is already 12.9 kWh into a fast charge at 88.5 kW
		energyReading(t0.Add(10*time.Minute), "IN_PROGRESS", "CONNECTED", 52, 43.1, 88.5, nil),
		energyReading(t0.Add(12*time.Minute), "IN_PROGRESS", "CONNECTED", 60, 50.0, 80, nil),
		energyReading(t0.Add(70*time.Minute), "COMPLETED", "CONNECTED", 100, 85.4, 0, map[string]any{"telemetry.metrics.xevBatteryChargerEnergyOutput.value": 57.6}),
	}
	for _, st := range steps {
		if err := d.Process(st); err != nil {
			t.Fatal(err)
		}
	}
	cs, _ := s.ListCharges(10)
	if len(cs) != 1 {
		t.Fatalf("one session: %d", len(cs))
	}
	c := cs[0]
	if c.StartSoC == nil || *c.StartSoC != 36 || c.StartKwh == nil || *c.StartKwh != 30.2 {
		t.Fatalf("starts from the level before the charge: %+v", c)
	}
	// 12.9 kWh at 88.5 kW is about 8.75 minutes before the first charging reading, so after the last parked one
	if min := (c.StartTS - t0.Unix()) / 60; min < 0 || min > 3 {
		t.Fatalf("start is estimated shortly after the last reading, not at the first charging one: %d min", min)
	}
	if c.ChargerKwh == nil || *c.ChargerKwh != 57.6 {
		t.Fatalf("the car's own count of what the charger delivered is kept: %+v", c.ChargerKwh)
	}
}

func TestACarsStaleChargerFigureIsNotBelieved(t *testing.T) {
	s, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	d := &Detector{S: s, TZ: time.UTC}
	t0 := time.Date(2026, 10, 20, 15, 34, 0, 0, time.UTC)
	stale := map[string]any{"telemetry.metrics.xevBatteryChargerEnergyOutput.value": 39.9} // from an earlier session
	steps := []*ford.State{
		energyReading(t0, "IN_PROGRESS", "CONNECTED", 99, 124.8, 11, stale),
		energyReading(t0.Add(2*time.Hour), "COMPLETED", "CONNECTED", 100, 129.8, 0, stale),
	}
	for _, st := range steps {
		if err := d.Process(st); err != nil {
			t.Fatal(err)
		}
	}
	cs, _ := s.ListCharges(10)
	if len(cs) != 1 || cs[0].ChargerKwh != nil {
		t.Fatalf("39.9 kWh cannot be what delivered 5 kWh to the battery: %+v", cs)
	}
}
