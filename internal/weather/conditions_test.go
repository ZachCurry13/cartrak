package weather

import "testing"

func drive(n int, speed float64) []Pt {
	out := make([]Pt, n)
	for i := range out {
		out[i] = Pt{TS: int64(i) * 60, Lat: 45 + float64(i)*0.001, Lon: -93, Speed: speed}
	}
	return out
}

func always(h Hour) func(float64, float64, int64) (Hour, bool) {
	return func(float64, float64, int64) (Hour, bool) { return h, true }
}

func TestParseReadsRainSnowAndTheWeatherCode(t *testing.T) {
	hs, err := Parse([]byte(`{"hourly":{"time":["2026-10-18T00:00"],"wind_speed_10m":[10],"wind_direction_10m":[90],"wind_gusts_10m":[20],"precipitation":[1.2],"snowfall":[0.3],"weather_code":[73]}}`))
	if err != nil || len(hs) != 1 || !hs[0].Wx || hs[0].RainMm != 1.2 || hs[0].SnowCm != 0.3 || hs[0].Code != 73 {
		t.Fatalf("%v %+v", err, hs)
	}
	old, _ := Parse([]byte(`{"hourly":{"time":["2026-10-18T00:00"],"wind_speed_10m":[10],"wind_direction_10m":[90]}}`))
	if len(old) != 1 || old[0].Wx {
		t.Fatalf("an answer with no rain figures is not marked as having them: %+v", old)
	}
}

func TestSummarizeLabelsTheWeatherAndPricesIt(t *testing.T) {
	cases := []struct {
		h     Hour
		label string
		pct   float64
	}{
		{Hour{Wx: true}, "Dry", 0},
		{Hour{Wx: true, RainMm: 0.3}, "Drizzle", 3},
		{Hour{Wx: true, RainMm: 1.5, Code: 61}, "Light rain", 5},
		{Hour{Wx: true, RainMm: 4}, "Rain", 8},
		{Hour{Wx: true, RainMm: 9}, "Heavy rain", 12},
		{Hour{Wx: true, RainMm: 0.8, SnowCm: 0.5, Code: 71}, "Snow", 8},
		{Hour{Wx: true, RainMm: 3, Code: 95}, "Thunderstorm", 8},
	}
	for _, c := range cases {
		got, ok := Summarize(drive(10, 80), always(c.h))
		if !ok || got.Label != c.label || got.BasePct != c.pct {
			t.Errorf("%+v: %+v ok=%v, want %q %v%%", c.h, got, ok, c.label, c.pct)
		}
	}
}

func TestSummarizeWeighsByMovingTimeAndIgnoresStopsAndMissingWeather(t *testing.T) {
	pts := drive(10, 80)
	for i := 5; i < 10; i++ { // the second half is a stop
		pts[i].Speed = 0
	}
	got, ok := Summarize(pts, func(_, _ float64, ts int64) (Hour, bool) {
		if ts < 120 {
			return Hour{Wx: true, RainMm: 3}, true
		}
		return Hour{Wx: true}, true
	})
	if !ok || got.Label != "Light rain" || got.WetShare != 0.5 {
		t.Fatalf("only the stretch before the stop counts and half of it is wet: %+v", got)
	}
	if _, ok := Summarize(drive(10, 80), func(float64, float64, int64) (Hour, bool) { return Hour{}, false }); ok {
		t.Fatal("no weather answer must not be reported as dry")
	}
	if _, ok := Summarize(drive(10, 80), always(Hour{RainMm: 3})); ok {
		t.Fatal("an hour saved before rain was asked for says nothing about rain")
	}
}

func TestEffectScalesWithWhatWasLearned(t *testing.T) {
	c, _ := Summarize(drive(10, 80), always(Hour{Wx: true, RainMm: 1.5}))
	if c.Effect(0) != 5 || c.Effect(1) != 5 || c.Effect(2) != 10 {
		t.Fatalf("%v %v %v", c.Effect(0), c.Effect(1), c.Effect(2))
	}
}

func TestLearnRainNeedsSeveralWetAndDryTripsAndMeasuresTheDifference(t *testing.T) {
	var few []RainSample
	for i := 0; i < 5; i++ {
		few = append(few, RainSample{RainMmH: 1, Ratio: 0.9}, RainSample{RainMmH: 0, Ratio: 1})
	}
	if _, ok := LearnRain(few); ok {
		t.Fatal("five of each is not enough")
	}
	var s []RainSample
	for i := 0; i < 8; i++ {
		s = append(s, RainSample{RainMmH: 1.2, Ratio: 0.9}, RainSample{RainMmH: 0, Ratio: 1.0}, RainSample{RainMmH: 0.2, Ratio: 5}) // a drizzle trip is neither wet nor dry
	}
	l, ok := LearnRain(s)
	if !ok || l.WetTrips != 8 || l.DryTrips != 8 || l.EffectPct < 11 || l.EffectPct > 11.2 || l.Scale < 2.2 || l.Scale > 2.3 {
		t.Fatalf("%+v ok=%v", l, ok)
	}
	var better []RainSample
	for i := 0; i < 8; i++ {
		better = append(better, RainSample{RainMmH: 1, Ratio: 1.2}, RainSample{RainMmH: 0, Ratio: 1})
	}
	if l, ok := LearnRain(better); !ok || l.EffectPct != 0 || l.Scale != 0.25 {
		t.Fatalf("rain never counts as helping: %+v", l)
	}
}
