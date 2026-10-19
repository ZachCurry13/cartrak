package tripwhy

import "testing"

func fp(v float64) *float64 { return &v }

func drive(odo0, kwh0, kph, perKm float64, n int, temp float64) []Reading {
	var out []Reading
	for i := 0; i <= n; i++ {
		d := kph / 3600 * 30 * float64(i)
		out = append(out, Reading{TS: int64(i) * 30, Odo: odo0 + d, Kwh: kwh0 - d*perKm, Speed: kph, OutsideC: fp(temp)})
	}
	return out
}

var base = Baseline{
	Bands:       []Band{{3, 100, 5.0}, {100, 400, 3.5}},
	Overall:     4.4,
	TempKmPKwh:  map[string]float64{"10 to 20 °C": 4.6, "Below freezing": 3.3},
	TempOverall: 4.4,
}

func TestAFastColdTripNamesSpeedAndTemperatureAsTheReasons(t *testing.T) {
	rs := drive(1000, 80, 120, 0.30, 120, -3) // 120 km/h in the cold at 0.30 kWh per km
	km := 120.0 / 3600 * 30 * 120
	r := Build(rs, 0.30*km, base, "Below freezing", nil)
	if !r.Ready || r.VsUsualPct < 20 {
		t.Fatalf("it used much more than usual: %+v", r)
	}
	keys := map[string]float64{}
	for _, f := range r.Factors {
		keys[f.Key] = f.Pct
	}
	if keys["speed"] <= 10 || keys["temperature"] <= 10 {
		t.Fatalf("speed and cold both cost: %+v", r.Factors)
	}
	if r.Facts.Share100 < 0.99 || r.Facts.AvgKph < 119 {
		t.Fatalf("%+v", r.Facts)
	}
}

func TestAGentleMildTripIsBetterThanUsualAndStopsAreCounted(t *testing.T) {
	rs := drive(2000, 80, 60, 0.15, 60, 15)
	rs = append(rs, Reading{TS: 2000, Odo: rs[len(rs)-1].Odo, Kwh: 70, Speed: 0}) // a stop (far later: skipped as a gap)
	km := 60.0 / 3600 * 30 * 60
	r := Build(rs, 0.15*km, base, "10 to 20 °C", nil)
	if !r.Ready || r.VsUsualPct >= 0 {
		t.Fatalf("%+v", r)
	}
	if r.Facts.Stops != 1 {
		t.Fatalf("one stop: %d", r.Facts.Stops)
	}
}

func TestTooLittleDrivingGivesNoExplanation(t *testing.T) {
	r := Build(drive(0, 80, 50, 0.2, 4, 10), 0.4, base, "", nil)
	if r.Ready {
		t.Fatalf("%+v", r)
	}
}

func TestTagsNameTheKeyReasons(t *testing.T) {
	// Freezing, fast, with hard braking and a long idle with the climate control on.
	var rs []Reading
	odo, kwh := 100.0, 80.0
	ts := int64(0)
	add := func(kph float64, dt int64, perKm float64, temp float64) {
		d := kph / 3600 * float64(dt)
		odo += d
		kwh -= d*perKm + 0.0 // energy for the distance
		ts += dt
		rs = append(rs, Reading{TS: ts, Odo: odo, Kwh: kwh, Speed: kph, OutsideC: fp(temp)})
	}
	add(0, 30, 0, -4)
	for i := 0; i < 8; i++ { // cruise, brake hard, accelerate again
		add(115, 30, 0.30, -4)
		add(115, 30, 0.30, -4)
		add(60, 30, 0.30, -4)
		add(115, 30, 0.30, -4)
	}
	for i := 0; i < 10; i++ { // sitting stopped with the heater on: 0.3 kWh each 30 s
		kwh -= 0.3
		ts += 30
		rs = append(rs, Reading{TS: ts, Odo: odo, Kwh: kwh, Speed: 0, OutsideC: fp(-4)})
	}
	r := Build(rs, 80-kwh, base, "Below freezing", nil)
	got := map[string]bool{}
	for _, tg := range r.Tags {
		got[tg.Key] = tg.Bad
	}
	for _, want := range []string{"cold", "speed", "idle", "brake", "launch"} {
		if !got[want] {
			t.Errorf("missing tag %q in %+v (facts %+v)", want, r.Tags, r.Facts)
		}
	}
	// A gentle, mild, efficient trip gets good words instead.
	good := drive(0, 80, 60, 0.12, 60, 15)
	km := 60.0 / 3600 * 30 * 60
	g := Build(good, 0.12*km, base, "10 to 20 °C", nil)
	if len(g.Tags) == 0 || g.Tags[0].Bad {
		t.Fatalf("%+v", g.Tags)
	}
}

func TestRegenerativeBrakingIsAGoodTagWhenTheCarReportsEnoughRangeGivenBack(t *testing.T) {
	km := 60.0 / 3600 * 30 * 60
	regen := km * 0.1 // a tenth of the distance given back
	r := Build(drive(0, 80, 60, 0.12, 60, 15), 0.12*km, base, "10 to 20 °C", &regen)
	found := false
	for _, tg := range r.Tags {
		if tg.Key == "regen" && !tg.Bad {
			found = true
		}
	}
	if !found || r.Facts.RegenKm == nil {
		t.Fatalf("%+v", r.Tags)
	}
	small := 0.1
	if r := Build(drive(0, 80, 60, 0.12, 60, 15), 0.12*km, base, "10 to 20 °C", &small); len(r.Tags) > 0 && r.Tags[0].Key == "regen" {
		t.Fatal("a little regen is not worth a tag")
	}
}
