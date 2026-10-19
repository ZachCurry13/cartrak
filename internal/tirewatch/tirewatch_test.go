package tirewatch

import "testing"

func f(v float64) *float64 { return &v }

func readings(days int, mod func(day int, p *[4]float64)) []Point {
	var out []Point
	for d := 0; d < days; d++ {
		p := [4]float64{240, 240, 240, 240}
		mod(d, &p)
		out = append(out, Point{TS: int64(d)*86400 + 40000, P: [4]*float64{f(p[0]), f(p[1]), f(p[2]), f(p[3])}, OutsideC: f(20)})
	}
	return out
}

func TestOneTireSlowlyLosingPressureIsFlaggedAndTheOthersAreNot(t *testing.T) {
	pts := readings(10, func(d int, p *[4]float64) { p[2] = 240 - 2.5*float64(d) })
	r := Build(pts, f(268.9), f(268.9))
	if !r.Tires[2].Leak || r.Tires[0].Leak || r.Tires[1].Leak || r.Tires[3].Leak {
		t.Fatalf("%+v", r.Tires)
	}
	if r.Note == "" || r.Tires[2].PerDay == nil || *r.Tires[2].PerDay > -2 {
		t.Fatalf("%+v %q", r.Tires[2].PerDay, r.Note)
	}
}

func TestAColdSnapThatLowersEveryTireIsNotALeak(t *testing.T) {
	var pts []Point
	for d := 0; d < 10; d++ { // pressure follows the temperature down together
		temp := 20 - float64(d)*2
		raw := 240 * (temp + 273.15) / 293.15
		pts = append(pts, Point{TS: int64(d)*86400 + 40000, P: [4]*float64{f(raw), f(raw), f(raw), f(raw)}, OutsideC: f(temp)})
	}
	r := Build(pts, nil, nil)
	for _, tr := range r.Tires {
		if tr.Leak {
			t.Fatalf("temperature, not a leak: %+v", r.Tires)
		}
	}
	if r.Tires[0].PerDay == nil || *r.Tires[0].PerDay > 0.1 || *r.Tires[0].PerDay < -0.1 {
		t.Fatalf("flat once scaled: %v", r.Tires[0].PerDay)
	}
}

func TestTooFewDaysGivesNoTrend(t *testing.T) {
	r := Build(readings(3, func(d int, p *[4]float64) { p[0] = 200 - 20*float64(d) }), nil, nil)
	if r.Tires[0].Leak || r.Tires[0].PerDay != nil || r.Tires[0].Kpa == nil {
		t.Fatalf("%+v", r.Tires[0])
	}
}
