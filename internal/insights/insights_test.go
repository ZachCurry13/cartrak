package insights

import (
	"math"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

func f(v float64) *float64 { return &v }

func trip(id int64, day time.Time, km, kwh float64) *store.Trip {
	end := day.Add(time.Hour).Unix()
	return &store.Trip{ID: id, StartTS: day.Unix(), EndTS: &end, DistanceKm: km, StartKwh: f(80), EndKwh: f(80 - kwh)}
}

func TestEfficiencyByMonthTemperatureAndBestAndWorstTrips(t *testing.T) {
	jan := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)
	jul := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	var ts []*store.Trip
	temps := map[int64]float64{}
	for i := int64(1); i <= 4; i++ { // winter: 100 km on 25 kWh, 4 km/kWh, at -5 degrees
		ts = append(ts, trip(i, jan.Add(time.Duration(i)*24*time.Hour), 100, 25))
		temps[i] = -5
	}
	for i := int64(5); i <= 8; i++ { // summer: 100 km on 16 kWh, 6.25 km/kWh, at 25 degrees
		ts = append(ts, trip(i, jul.Add(time.Duration(i)*24*time.Hour), 100, 16))
		temps[i] = 25
	}
	ts = append(ts, trip(9, jul, 2, 1), &store.Trip{ID: 10, Open: true}) // too short, and unfinished
	rep := Build(ts, nil, temps, time.UTC)
	if rep.Trips != 8 || len(rep.Months) != 2 || rep.Months[0].Month != "2026-07" {
		t.Fatalf("%+v", rep)
	}
	if got := rep.Months[1].KmPerKwh; math.Abs(got-4) > 1e-9 || rep.Months[1].AvgTempC == nil || *rep.Months[1].AvgTempC != -5 {
		t.Fatalf("January: %+v", rep.Months[1])
	}
	if len(rep.Bands) != 2 || rep.Bands[0].Name != "Below freezing" || math.Abs(rep.Bands[0].KmPerKwh-4) > 1e-9 || math.Abs(rep.Bands[1].KmPerKwh-6.25) > 1e-9 {
		t.Fatalf("bands: %+v", rep.Bands)
	}
	if len(rep.Best) != 3 || len(rep.Worst) != 3 || rep.Best[0].KmPerKwh < 6 || rep.Worst[0].KmPerKwh > 4.1 {
		t.Fatalf("best %+v worst %+v", rep.Best, rep.Worst)
	}
	few := Build(ts[:3], nil, temps, time.UTC)
	if len(few.Best) != 0 {
		t.Fatal("too few trips to rank")
	}
}
