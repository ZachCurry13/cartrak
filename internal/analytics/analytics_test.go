package analytics

import (
	"math"
	"testing"
	"time"
)

var utc = time.UTC

func at(s string) int64     { t, _ := time.Parse("2006-01-02 15:04", s); return t.Unix() }
func fp(v float64) *float64 { return &v }

func TestChargingSplitsHomeAndPublicByMonth(t *testing.T) {
	cs := []Charge{
		{ID: 1, Start: at("2026-09-03 22:00"), Kwh: 30, WallKwh: 33, Cost: fp(4.95), Where: "home", PeakKw: 11, Eff: fp(0.9)},
		{ID: 2, Start: at("2026-10-01 13:00"), Kwh: 55, WallKwh: 57.5, Cost: fp(31), Receipt: true, Where: "public", Place: "Red charger", PeakKw: 150},
		{ID: 3, Start: at("2026-10-02 22:30"), Kwh: 20, Where: "home", PeakKw: 7},
		{ID: 4, Start: at("2026-10-05 09:00"), Kwh: 0, Where: "home"}, // nothing added: left out
	}
	r := Charging(cs, utc)
	if r.Sessions != 3 || r.Kwh != 105 || math.Abs(r.Cost-35.95) > 1e-9 {
		t.Fatalf("totals: %+v", r)
	}
	if len(r.Months) != 2 || r.Months[0].Month != "2026-09" || r.Months[1].PublicKwh != 55 || r.Months[1].HomeKwh != 20 {
		t.Fatalf("months oldest first, home and public apart: %+v", r.Months)
	}
	if math.Abs(r.Months[1].PricePerKwh-31/57.5) > 1e-9 || r.Months[0].Eff != 0.9 {
		t.Fatalf("price from priced sessions only, efficiency measured: %+v", r.Months)
	}
	if r.Places[0].Name != "Red charger" || r.Places[1].Name != "Home" || r.Places[1].Sessions != 2 {
		t.Fatalf("places by energy: %+v", r.Places)
	}
	if r.PlugIn[int(time.Thursday)][22] != 1 || len(r.Fastest) != 1 || r.Fastest[0].ID != 2 {
		t.Fatalf("plug-in hours and fast charges: %v %+v", r.PlugIn[4], r.Fastest)
	}
	if math.Abs(r.HomeShare-50.0/105) > 1e-9 {
		t.Fatalf("home share: %v", r.HomeShare)
	}
}

func TestDrivingFindsTheTemperatureLineAndPatterns(t *testing.T) {
	var ts []Trip
	for i := 0; i < 10; i++ {
		temp := float64(-5 + i*3) // -5 to 22 °C
		eff := 4 + 0.05*temp      // 0.05 km/kWh better per degree
		start := at("2026-10-01 08:00") + int64(i)*86400
		ts = append(ts, Trip{ID: int64(i + 1), Start: start, End: start + 1800, Km: 20, Kwh: 20 / eff, TempC: fp(temp)})
	}
	ts = append(ts, Trip{ID: 99, Start: at("2026-10-01 17:00"), End: at("2026-10-01 17:05"), Km: 1})
	r := Driving(ts, utc)
	if r.Trips != 11 || len(r.Temp) != 10 || r.TempFit == nil || math.Abs(r.TempFit[1]-0.05) > 1e-6 || math.Abs(r.TempFit[0]-4) > 1e-6 {
		t.Fatalf("temperature fit: %+v %v", r.TempFit, len(r.Temp))
	}
	if r.Lengths[0].Trips != 1 || r.Lengths[4].Trips != 10 {
		t.Fatalf("lengths: %+v", r.Lengths)
	}
	if len(r.Days) != 10 || r.Days[0].Km != 21 || r.Days[0].Trips != 2 {
		t.Fatalf("days: %+v", r.Days[0])
	}
	if len(r.Speed) != 1 || r.Speed[0].FromKph != 30 || r.Speed[0].Trips != 10 {
		t.Fatalf("40 km/h trips: %+v", r.Speed)
	}
	if r.HourWeek[int(time.Thursday)][8] != 40 { // Oct 1 and Oct 8
		t.Fatal("km by hour of the week")
	}
	if Driving(ts[:5], utc).TempFit != nil {
		t.Fatal("too few trips for a line")
	}
}

func TestAnUnbelievableDriveIsNotTheEfficiencyRecord(t *testing.T) {
	var ts []Trip
	for i := int64(1); i <= 6; i++ { // a car that gets about 5.6 km/kWh (3.5 mi/kWh)
		ts = append(ts, Trip{ID: i, Start: i * 86400, Km: 28, Kwh: 5})
	}
	ts = append(ts, Trip{ID: 7, Start: 7 * 86400, Km: 30, Kwh: 2.9}) // 10.3 km/kWh: a stale end reading
	ts = append(ts, Trip{ID: 8, Start: 8 * 86400, Km: 30, Kwh: 4.2}) // 7.1 km/kWh: a good, believable drive
	for _, r := range Records(ts, nil, time.UTC) {
		if r.Key == "efficient" && r.TripID != 8 {
			t.Fatalf("the best believable drive wins: %+v", r)
		}
	}
}

func TestRecordsAndStreak(t *testing.T) {
	ts := []Trip{
		{ID: 1, Start: at("2026-10-01 08:00"), End: at("2026-10-01 09:00"), Km: 80, Kwh: 20, TempC: fp(-3), MaxKph: 120},
		{ID: 2, Start: at("2026-10-02 08:00"), End: at("2026-10-02 08:30"), Km: 20, Kwh: 3, TempC: fp(18), MaxKph: 90},
		{ID: 3, Start: at("2026-10-03 08:00"), End: at("2026-10-03 08:10"), Km: 5, TempC: fp(25)},
		{ID: 4, Start: at("2026-10-07 08:00"), End: at("2026-10-07 08:10"), Km: 5},
	}
	cs := []Charge{{ID: 7, Start: at("2026-10-01 12:00"), Kwh: 55, WallKwh: 57, Cost: fp(31), PeakKw: 150, Where: "public"}, {ID: 8, Start: at("2026-10-02 22:00"), Kwh: 30, Cost: fp(9), PeakKw: 11, Where: "public"},
		{ID: 9, Start: at("2026-10-03 22:00"), Kwh: 30, Cost: fp(3), Where: "home"}, {ID: 10, Start: at("2026-10-04 12:00"), Kwh: 20, Cost: fp(0), Where: "public"}}
	got := map[string]Record{}
	for _, r := range Records(ts, cs, utc) {
		got[r.Key] = r
	}
	if got["longest"].TripID != 1 || got["efficient"].TripID != 2 || got["topSpeed"].Value != 120 || got["coldest"].Value != -3 || got["hottest"].TripID != 3 {
		t.Fatalf("drive records: %+v", got)
	}
	if got["peakKw"].ChargeID != 7 || got["bigCharge"].ChargeID != 7 || got["cheapest"].ChargeID != 8 {
		t.Fatalf("charge records: %+v", got)
	}
	if got["streak"].Value != 3 || got["streak"].Date != "2026-10-03" || got["bigDay"].Date != "2026-10-01" {
		t.Fatalf("days: %+v %+v", got["streak"], got["bigDay"])
	}
}

func TestMonthRecapWithSavings(t *testing.T) {
	ts := []Trip{
		{ID: 1, Start: at("2026-10-01 08:00"), End: at("2026-10-01 09:00"), Km: 100, Kwh: 20},
		{ID: 2, Start: at("2026-10-01 18:00"), End: at("2026-10-01 18:30"), Km: 20, Kwh: 4},
		{ID: 3, Start: at("2026-09-20 08:00"), End: at("2026-09-20 09:00"), Km: 50, Kwh: 10},
	}
	cs := []Charge{{Start: at("2026-10-02 22:00"), Kwh: 24, Cost: fp(3.6), Where: "home"}}
	r := PeriodRecap("2026-10", ts, cs, Gas{LitersPerKm: 0.08, PricePerLiter: 1, GridKgPerKwh: 0.4}, utc)
	if r.Trips != 2 || r.Km != 120 || r.DaysDriven != 1 || r.Longest.ID != 1 || *r.PrevKm != 50 || r.KmPerKwh != 5 || r.HomeShare != 1 {
		t.Fatalf("recap: %+v", r)
	}
	// gas: 120 km x 0.08 L x 1 = 9.60; electricity for 24 kWh at 0.15 = 3.60; saved 6.00
	if math.Abs(*r.GasCost-9.6) > 1e-9 || math.Abs(*r.Saved-6) > 1e-9 || math.Abs(*r.KgCO2Saved-(9.6*2.348-24*0.4)) > 1e-9 {
		t.Fatalf("savings: %v %v %v", *r.GasCost, *r.Saved, *r.KgCO2Saved)
	}
	if PeriodRecap("2026-10", ts, cs, Gas{}, utc).Saved != nil {
		t.Fatal("no gas car, no savings")
	}
}

func TestBatteryTimeAtEachLevel(t *testing.T) {
	base := at("2026-10-01 00:00")
	rs := []Reading{{base, 85}, {base + 3600, 85}, {base + 7200, 50}, {base + 10800, 15}, {base + 10800 + 10*86400, 95}}
	r := Battery(rs, utc)
	// 2 h at 85 (80-90), 1 h at 50 (50-80), then 6 h capped at 15 (0-20)
	if math.Abs(r.Hours-9) > 1e-9 || math.Abs(r.LowShare-6.0/9) > 1e-9 || r.HighShare != 0 {
		t.Fatalf("bands: %+v", r)
	}
	if len(r.Days) != 2 || r.Days[0].Min != 15 || r.Days[0].Max != 85 || r.Days[0].End != 15 {
		t.Fatalf("days: %+v", r.Days)
	}
	if Battery(nil, utc).Hours != 0 {
		t.Fatal("empty")
	}
}

func TestATripWithACorrectedDistanceIsNotTheEfficiencyRecord(t *testing.T) {
	var ts []Trip
	for i := int64(1); i <= 6; i++ {
		ts = append(ts, Trip{ID: i, Start: i * 86400, Km: 28, Kwh: 5})
	}
	// the odometer said 9 km on 1.9 kWh; the distance was later corrected to 18 km, which doubles the efficiency
	ts = append(ts, Trip{ID: 7, Start: 7 * 86400, Km: 18, Kwh: 1.9, Corrected: true})
	for _, r := range Records(ts, nil, time.UTC) {
		if r.Key == "efficient" && r.TripID == 7 {
			t.Fatalf("a corrected distance is not the odometer's: %+v", r)
		}
	}
}
