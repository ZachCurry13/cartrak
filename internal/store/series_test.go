package store

import (
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
)

func TestDailySeriesAveragesPerDayAndKeepsNewColumns(t *testing.T) {
	s := openTemp(t)
	day := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	v := func(x float64) *float64 { return &x }
	for i, volts := range []float64{12.4, 12.8, 12.6} {
		st := &ford.State{TS: day.Add(time.Duration(i) * time.Minute), Battery12V: v(volts), OilLifePct: v(60), Raw: []byte(`{}`)}
		if _, err := s.SaveState("test", st); err != nil {
			t.Fatal(err)
		}
	}
	next := &ford.State{TS: day.Add(24 * time.Hour), Battery12V: v(11.9), FuelPct: v(40), HVVolts: v(371), CapacityKwh: v(88), Raw: []byte(`{}`)}
	if _, err := s.SaveState("test", next); err != nil {
		t.Fatal(err)
	}

	days, err := s.DailySeries("battery12v", 0)
	if err != nil || len(days) != 2 {
		t.Fatalf("days = %+v, %v", days, err)
	}
	if d := days[0]; d.N != 3 || d.Min != 12.4 || d.Max != 12.8 || d.Avg < 12.59 || d.Avg > 12.61 {
		t.Fatalf("first day: %+v", d)
	}
	if oil, _ := s.DailySeries("oilLifePct", 0); len(oil) != 1 {
		t.Fatalf("the oil series must skip days with no oil reading: %+v", oil)
	}
	if later, _ := s.DailySeries("battery12v", day.Add(12*time.Hour).Unix()); len(later) != 1 || later[0].Min != 11.9 {
		t.Fatalf("the from limit: %+v", later)
	}
	if _, err := s.DailySeries("tire_fl; DROP TABLE samples", 0); err == nil {
		t.Fatal("only the listed series may be asked for")
	}
	x, _ := s.LatestSample()
	if x.FuelPct == nil || *x.FuelPct != 40 || x.HVVolts == nil || *x.HVVolts != 371 || x.CapacityKwh == nil || *x.CapacityKwh != 88 {
		t.Fatalf("new columns did not round trip: %+v", x)
	}
}
