package analytics

import (
	"math"
	"testing"
)

func TestPersonalitiesRankTheBestFitFirst(t *testing.T) {
	var cs []Charge
	for i := 0; i < 10; i++ {
		cs = append(cs, Charge{Start: at("2026-10-01 22:00") + int64(i)*86400, Kwh: 20, Where: "home", PeakKw: 11})
	}
	var ts []Trip
	for i := 0; i < 10; i++ {
		s := at("2026-10-05 07:30") + int64(i%5)*86400 // Mon to Fri mornings
		ts = append(ts, Trip{Start: s, End: s + 1200, Km: 15})
	}
	p := Personalities(ts, cs, utc)
	if len(p) < 2 || p[0].Key != "night" || p[0].Share != 1 {
		t.Fatalf("all charging at night wins: %+v", p)
	}
	found := false
	for _, x := range p {
		found = found || x.Key == "commuter"
	}
	if !found {
		t.Fatalf("weekday mornings make a commuter: %+v", p)
	}
	if p := Personalities([]Trip{{Start: at("2026-10-03 12:00"), End: at("2026-10-03 12:20"), Km: 12}}, nil, utc); len(p) != 1 || p[0].Key != "steady" {
		t.Fatalf("too little to say: %+v", p)
	}
}

func TestEnergyFlowFollowsEveryKwh(t *testing.T) {
	from, to := at("2026-10-01 00:00"), at("2026-11-01 00:00")
	cs := []Charge{{Start: at("2026-10-02 22:00"), Kwh: 45, WallKwh: 50, Where: "home"}, {Start: at("2026-10-10 12:00"), Kwh: 30, Where: "public"}, {Start: at("2026-09-30 12:00"), Kwh: 99}}
	ts := []Trip{
		{Start: at("2026-10-03 08:00"), End: at("2026-10-03 08:30"), Km: 20, Kwh: 5},   // 40 km/h: city
		{Start: at("2026-10-04 08:00"), End: at("2026-10-04 09:00"), Km: 110, Kwh: 25}, // 110 km/h: highway
	}
	f := EnergyFlow(ts, cs, from, to, 10)
	if f.HomeKwh != 50 || f.PublicKwh != 30 || f.LostKwh != 5 || f.CityKwh != 5 || f.HighwayKwh != 25 || f.CityKm != 20 {
		t.Fatalf("flow: %+v", f)
	}
	// charged 75 into the battery: 30 driven, 10 still there, 35 the rest
	if math.Abs(f.OtherKwh-35) > 1e-9 {
		t.Fatalf("the rest: %v", f.OtherKwh)
	}
}

func TestTypicalDay(t *testing.T) {
	from, to := at("2026-10-05 00:00"), at("2026-10-15 00:00") // ten days from a Monday
	var ts []Trip
	var cs []Charge
	for i := int64(0); i < 10; i++ {
		ts = append(ts, Trip{Start: from + i*86400 + 7*3600 + 15*60, End: from + i*86400 + 7*3600 + 45*60, Km: 20})
		cs = append(cs, Charge{Start: from + i*86400 + 21*3600 + 30*60, End: from + i*86400 + 23*3600 + 30*60, Kwh: 10})
	}
	r := TypicalDay(ts, cs, from, to, utc)
	if r.Days != 10 || math.Abs(r.Driving[29]-1) > 1e-9 || r.Driving[40] != 0 || math.Abs(r.Charging[88]-1) > 1e-9 {
		t.Fatalf("every day at 7:15 and 22:00: %v %v", r.Driving[29], r.Charging[88])
	}
	if r.LeaveAt != "07:15" || r.PlugInAt != "21:30" {
		t.Fatalf("usual times: %q %q", r.LeaveAt, r.PlugInAt)
	}
	if math.Abs(r.Parked-(1-2.5/24)) > 1e-9 {
		t.Fatalf("parked share: %v", r.Parked)
	}
	// the same ten days inside a whole year: only the days since the first one count
	if y := TypicalDay(ts, cs, from-300*86400, to, utc); y.Days != 10 || math.Abs(y.Driving[29]-1) > 1e-9 {
		t.Fatalf("a year that began before the data: %d days, %v", y.Days, y.Driving[29])
	}
}

func TestBadgesAreEarnedWhenTheGoalIsReached(t *testing.T) {
	var ts []Trip
	for i := int64(0); i < 31; i++ { // 31 days in a row, 60 km a day, the last one a long, cold, efficient drive
		s := at("2026-10-01 08:00") + i*86400
		ts = append(ts, Trip{ID: i + 1, Start: s, End: s + 3600, Km: 60, Kwh: 10}) // 6 km/kWh, so 9.4 on the last is believable
	}
	ts[30].Km, ts[30].Kwh, ts[30].TempC = 170, 18, fp(-20)
	cs := []Charge{{Start: at("2026-10-02 23:00"), End: at("2026-10-03 01:00"), Kwh: 60, PeakKw: 155}}
	got := map[string]Badge{}
	for _, b := range Badges(ts, cs, map[string]int64{"A": 1, "B": 2}, utc) {
		got[b.Key] = b
	}
	if got["streak30"].Earned != ts[29].End || got["month1000"].Earned != ts[26].End {
		t.Fatalf("streak on day 30, 1,000 miles on day 27: %+v %+v", got["streak30"], got["month1000"])
	}
	if got["cold"].Earned != ts[30].Start || got["road"].Earned == 0 || got["kw150"].Earned == 0 || got["hyper"].Earned == 0 {
		t.Fatalf("one-off badges: %+v", got)
	}
	if got["hot"].Earned != 0 || got["hot"].Progress() != 0 || got["night50"].Value != 1 || math.Abs(got["towns25"].Progress()-2.0/25) > 1e-9 {
		t.Fatalf("progress: %+v %+v %+v", got["hot"], got["night50"], got["towns25"])
	}
	if got["km16000"].Earned != 0 || math.Abs(got["km16000"].Progress()-(30*60+170)/16093.0) > 1e-9 {
		t.Fatalf("10,000 miles not yet: %+v", got["km16000"])
	}
}
