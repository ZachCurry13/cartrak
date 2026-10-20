package planahead

import (
	"math"
	"testing"
	"time"
)

// history builds 28 completed days ending the day before today, with the given distance for each weekday.
func history(today time.Time, perWeekday map[time.Weekday]float64) []DayKm {
	var out []DayKm
	for i := 28; i >= 1; i-- {
		d := today.AddDate(0, 0, -i)
		out = append(out, DayKm{Date: d.Format("2006-01-02"), Km: perWeekday[d.Weekday()]})
	}
	return out
}

func TestEachDayIsForecastFromTheSameWeekdayAndTheBatteryRunsDown(t *testing.T) {
	today := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC) // a Tuesday
	km := map[time.Weekday]float64{time.Monday: 40, time.Tuesday: 10, time.Wednesday: 50, time.Thursday: 10, time.Friday: 60, time.Saturday: 0, time.Sunday: 0}
	f, ok := Build(Input{History: history(today, km), Today: today, KwhPerKm: 0.25, CapacityKwh: 80, StartPct: 50, ReservePct: 20})
	if !ok || len(f.Days) != 7 {
		t.Fatalf("a week: %+v %v", f, ok)
	}
	if f.Days[0].Weekday != "Wednesday" || f.Days[0].Km != 50 || f.Days[2].Km != 60 {
		t.Fatalf("the same weekday in the history: %+v", f.Days)
	}
	// 50 km is 12.5 kWh of 80, so 15.625 points
	if math.Abs(f.Days[0].EndPct-(50-12.5/80*100)) > 0.01 {
		t.Fatalf("battery after the first day: %v", f.Days[0].EndPct)
	}
	// Wed 50 + Thu 10 + Fri 60 = 120 km = 30 kWh = 37.5 points: 50 -> 12.5, which is below 20 on Friday
	if f.ChargeBy != "2026-10-23" || f.ChargeByDay != "Friday" {
		t.Fatalf("charge by: %q %q", f.ChargeBy, f.ChargeByDay)
	}
	if math.Abs(f.TotalKm-(50+10+60+0+0+40+10)) > 0.01 {
		t.Fatalf("total km: %v", f.TotalKm)
	}
}

func TestNoChargeIsNeededWhenTheWeekFits(t *testing.T) {
	today := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	km := map[time.Weekday]float64{time.Monday: 5, time.Tuesday: 5, time.Wednesday: 5, time.Thursday: 5, time.Friday: 5, time.Saturday: 5, time.Sunday: 5}
	f, ok := Build(Input{History: history(today, km), Today: today, KwhPerKm: 0.2, CapacityKwh: 80, StartPct: 80, ReservePct: 20})
	if !ok || f.ChargeBy != "" || f.EndPct < 70 {
		t.Fatalf("plenty left: %+v", f)
	}
}

func TestTooLittleHistoryOrNoEnergyFigureGivesNoForecast(t *testing.T) {
	today := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	few := history(today, map[time.Weekday]float64{})[:10]
	if _, ok := Build(Input{History: few, Today: today, KwhPerKm: 0.2, CapacityKwh: 80, StartPct: 80}); ok {
		t.Fatal("ten days is not enough")
	}
	if _, ok := Build(Input{History: history(today, map[time.Weekday]float64{}), Today: today, KwhPerKm: 0, CapacityKwh: 80}); ok {
		t.Fatal("without kWh per km there is nothing to forecast")
	}
}

func TestASteadyDailyDistanceGivesASteadyForecast(t *testing.T) {
	today := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	h := history(today, map[time.Weekday]float64{time.Monday: 28, time.Tuesday: 28, time.Wednesday: 28, time.Thursday: 28, time.Friday: 28, time.Saturday: 28, time.Sunday: 28})
	h = h[:14] // two weeks: each weekday appears twice
	f, ok := Build(Input{History: h, Today: today, KwhPerKm: 0.2, CapacityKwh: 80, StartPct: 80, ReservePct: 20})
	if !ok || f.Days[0].Km != 28 {
		t.Fatalf("a steady 28 km a day: %+v", f.Days[0])
	}
}
