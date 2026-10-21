package analytics

import (
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/fuel"
)

func byKey(bs []Badge) map[string]Badge {
	m := map[string]Badge{}
	for _, b := range bs {
		m[b.Key] = b
	}
	return m
}

func tm(s string) int64 {
	t, err := time.ParseInLocation("2006-01-02 15:04", s, time.UTC)
	if err != nil {
		panic(err)
	}
	return t.Unix()
}

func TestTrophiesFromHabits(t *testing.T) {
	trip := func(id int64, start, end string, km float64, from, to string) Trip {
		return Trip{ID: id, Start: tm(start), End: tm(end), Km: km, From: from, To: to}
	}
	ts := []Trip{
		trip(1, "2026-01-01 05:00", "2026-01-01 09:00", 300, "Home", "Cabin"), // a holiday, before 6 am
		trip(2, "2026-01-01 16:00", "2026-01-01 20:00", 280, "Cabin", "Home"), // 580 km that day, back where it began
		trip(3, "2026-07-04 22:30", "2026-07-05 00:30", 40, "Home", "Gym"),    // ends after midnight; July 4
		trip(4, "2026-10-31 10:00", "2026-10-31 10:20", 5, "", ""),
	}
	ts[0].OdoKm = 80500 // past 50,000 miles
	ts = append(ts, trip(5, "2026-12-25 09:00", "2026-12-25 09:30", 10, "Home", "Gym"), trip(6, "2026-12-25 12:00", "2026-12-25 12:30", 10, "Gym", "Mall"), trip(7, "2026-12-25 15:00", "2026-12-25 15:30", 10, "Mall", "Church"))
	got := byKey(MoreBadges(ts, nil, nil, time.UTC))
	for _, k := range []string{"marathon", "nightrider", "sunrise", "round", "holiday", "odo50", "triple"} {
		if k == "sunrise" {
			if got[k].Earned != 0 || got[k].Value != 1 {
				t.Errorf("sunrise needs 10 days, has one: %+v", got[k])
			}
			continue
		}
		if k == "holiday" {
			if got[k].Earned != tm("2026-12-25 09:30") || got[k].Value != 4 {
				t.Errorf("all four holidays: %+v", got[k])
			}
			continue
		}
		if got[k].Earned == 0 {
			t.Errorf("%s should be earned: %+v", k, got[k])
		}
	}
	if got["odo100"].Earned != 0 || got["regular"].Earned != 0 || got["weekend"].Earned != 0 {
		t.Errorf("not yet: %+v %+v %+v", got["odo100"], got["regular"], got["weekend"])
	}
}

func TestEfficiencyTrophiesIgnoreCorrectedDistancesAndNeedAnAverageToBeat(t *testing.T) {
	var ts []Trip
	for i := int64(0); i < 12; i++ { // a car that gets 5 km/kWh on drives of 20 km in the cold
		c := -10.0
		ts = append(ts, Trip{ID: i + 1, Start: tm("2026-01-10 08:00") + i*86400, End: tm("2026-01-10 08:30") + i*86400, Km: 20, Kwh: 4, TempC: &c})
	}
	c := -10.0
	ts = append(ts, Trip{ID: 99, Start: tm("2026-02-01 08:00"), End: tm("2026-02-01 08:30"), Km: 20, Kwh: 2, TempC: &c, Corrected: true})
	if got := byKey(MoreBadges(ts, nil, nil, time.UTC))["winter"]; got.Earned != 0 {
		t.Fatalf("a corrected distance does not earn it, and the others only match the average: %+v", got)
	}
	ts[len(ts)-1].Corrected = false
	if got := byKey(MoreBadges(ts, nil, nil, time.UTC))["winter"]; got.Earned == 0 {
		t.Fatalf("a cold drive that beat the average earns it: %+v", got)
	}
}

func TestChargingAndFuelTrophies(t *testing.T) {
	eff := 0.9
	var cs []Charge
	for d := 0; d < 14; d++ { // a charge every evening for two weeks
		s := tm("2026-03-01 19:00") + int64(d)*86400
		c := Charge{Start: s, End: s + 3600, Kwh: 10, Where: "home"}
		if d < 10 {
			c.Eff = &eff
		}
		cs = append(cs, c)
	}
	cs = append(cs, Charge{Start: tm("2026-04-01 12:00"), End: tm("2026-04-01 12:25"), Kwh: 45, Where: "public"})
	f := func(v float64) *float64 { return &v }
	fills := []fuel.Fill{
		{TS: 1, OdoKm: f(1000), Gallons: f(10), PricePerGal: f(3.50)}, {TS: 2, OdoKm: f(1300), Gallons: f(10), PricePerGal: f(3.50)},
		{TS: 3, OdoKm: f(1600), Gallons: f(10), PricePerGal: f(3.50)}, {TS: 4, OdoKm: f(1900), Gallons: f(10), PricePerGal: f(3.50)},
		{TS: 5, OdoKm: f(2200), Gallons: f(10), PricePerGal: f(3.50)}, {TS: 6, OdoKm: f(3000), Gallons: f(10), PricePerGal: f(3.00)},
	}
	got := byKey(MoreBadges(nil, cs, fills, time.UTC))
	if got["quick"].Earned == 0 || got["detective"].Earned == 0 || got["plugstreak"].Earned == 0 || got["muncher"].Earned != 6 || got["thrifty"].Earned != 6 {
		t.Fatalf("quick, detective, 14 evenings, 800 km on a tank and a cheap fill-up: %+v", got)
	}
}

func TestOdometerTrophiesAlreadyPassedSayAndTheNextOneHasAnEstimate(t *testing.T) {
	now := tm("2026-10-21 12:00")
	var ts []Trip
	for d := 0; d < 30; d++ { // 50 km a day for a month, the odometer already at 88,000 miles when it began
		s := now - int64(30-d)*86400
		ts = append(ts, Trip{ID: int64(d + 1), Start: s, End: s + 1800, Km: 50, OdoKm: 88000*kmPerMile + float64(d+1)*50})
	}
	bs := byKey(WithEstimates(MoreBadges(ts, nil, nil, time.UTC), ts, time.Unix(now, 0)))
	if bs["odo50"].Earned == 0 || bs["odo50"].Note == "" || bs["odo100"].Earned != 0 || bs["odo100"].Note != "" {
		t.Fatalf("50,000 is behind it, 100,000 is not: %+v %+v", bs["odo50"], bs["odo100"])
	}
	km100 := 100000 * kmPerMile
	want := now + int64((km100-(88000*kmPerMile+30*50))/50*86400)
	if got := bs["odo100"].Eta; got < want-86400 || got > want+86400 {
		t.Fatalf("about when 100,000 miles comes at 50 km a day: %d, want about %d", got, want)
	}
	if bs["odo50"].Eta != 0 {
		t.Fatal("an earned trophy needs no estimate")
	}
	if got := byKey(WithEstimates(MoreBadges(ts[:3], nil, nil, time.UTC), ts[:3], time.Unix(now, 0)))["odo100"].Eta; got != 0 {
		t.Fatal("under a week of driving is too little to estimate from")
	}
}
