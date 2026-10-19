package windcalc

import (
	"math"
	"testing"

	"github.com/ZachCurry13/cartrak/internal/weather"
)

// eastbound drives due east at 100 km/h for n readings 30 s apart near 45 degrees north.
func eastbound(n int) []Point {
	var ps []Point
	for i := 0; i < n; i++ {
		ps = append(ps, Point{TS: int64(i) * 30, Lat: 45, Lon: -93 + float64(i)*0.00176, Speed: 100})
	}
	return ps
}

func wind(from, kph float64) Lookup {
	return func(lat, lon float64, ts int64) (weather.Hour, bool) {
		return weather.Hour{SpeedKm: kph, DirDeg: from, GustKm: kph * 1.5}, true
	}
}

func TestWindFromTheWestIsATailwindForAnEastboundCar(t *testing.T) {
	s, ok := Estimate(eastbound(60), wind(270, 25))
	if !ok || s.HeadKph > -24 || s.HeadKph < -26 || s.EffectPct >= 0 || s.CrossKph > 1 {
		t.Fatalf("a 25 km/h tailwind helps: %+v", s)
	}
	if s.GustKph < 37 {
		t.Fatalf("gusts: %+v", s)
	}
}

func TestWindFromTheEastIsAHeadwindAndCostsEnergy(t *testing.T) {
	s, ok := Estimate(eastbound(60), wind(90, 25))
	if !ok || s.HeadKph < 24 || s.EffectPct < 10 {
		t.Fatalf("a 25 km/h headwind at 100 km/h costs well over ten percent of the drag: %+v", s)
	}
}

func TestWindFromTheNorthIsAllCrosswindAndTouchesLittle(t *testing.T) {
	s, ok := Estimate(eastbound(60), wind(0, 30))
	if !ok || math.Abs(s.HeadKph) > 1 || s.CrossKph < 29 || math.Abs(s.EffectPct) > 1 {
		t.Fatalf("%+v", s)
	}
}

func TestNoWindDataOrTooLittleDrivingGivesNothing(t *testing.T) {
	if _, ok := Estimate(eastbound(60), func(float64, float64, int64) (weather.Hour, bool) { return weather.Hour{}, false }); ok {
		t.Fatal("no wind reading, no estimate")
	}
	if _, ok := Estimate(eastbound(3), wind(90, 20)); ok {
		t.Fatal("under 3 km")
	}
	slow := eastbound(60)
	for i := range slow {
		slow[i].Speed = 10
	}
	if _, ok := Estimate(slow, wind(90, 20)); ok {
		t.Fatal("slow driving is skipped")
	}
}
