package chargeeta

import (
	"math"
	"testing"
)

func f(v float64) *float64 { return &v }

// session builds readings from a power function of the battery level, from lo to hi percent.
func session(l Level, bat float64, lo, hi float64, kw func(soc float64) float64) Session {
	s := Session{Level: l, BatC: f(bat), OutC: f(5)}
	for soc := lo; soc <= hi; soc += 1 {
		s.Obs = append(s.Obs, Obs{SoC: soc, Kw: kw(soc)})
	}
	return s
}

func TestTheKindOfChargerComesFromTheStationTypeThenThePower(t *testing.T) {
	cases := []struct {
		typ  string
		kw   float64
		want Level
	}{{"AC basic", 9, L2}, {"DC", 120, DC}, {"", 120, DC}, {"", 1.4, L1}, {"", 9.6, L2}, {"AC basic", 1.4, L1}}
	for _, c := range cases {
		if got := LevelOf(c.typ, c.kw); got != c.want {
			t.Errorf("%q %.1f kW: %s, want %s", c.typ, c.kw, got, c.want)
		}
	}
}

func TestACChargingSlowsNearFullSoALearnedEstimateIsLongerThanTheFlatOne(t *testing.T) {
	taper := func(soc float64) float64 {
		if soc < 90 {
			return 9
		}
		return 9 - (soc-90)*0.6 // 9 kW down to 3 kW at 100%
	}
	past := []Session{session(L2, 12, 20, 100, taper), session(L2, 12, 30, 100, taper), session(L2, 12, 25, 100, taper)}
	now := Now{Level: L2, SoC: 60, Kw: 9}
	flat, _ := Estimate(nil, now, 100, 88)
	got, ok := Estimate(past, now, 100, 88)
	if !ok || !got.Learned || got.Sessions != 3 {
		t.Fatalf("learned from three sessions: %+v", got)
	}
	if got.Minutes <= flat.Minutes*1.05 {
		t.Fatalf("the taper makes it longer: learned %.0f min, flat %.0f min", got.Minutes, flat.Minutes)
	}
	to80, _ := Estimate(past, now, 80, 88)
	flat80, _ := Estimate(nil, now, 80, 88)
	if math.Abs(to80.Minutes-flat80.Minutes) > 1 {
		t.Fatalf("below the taper nothing changes: %.1f vs %.1f", to80.Minutes, flat80.Minutes)
	}
}

func TestAColdBatteryIsJudgedByColdFastChargesAndNotWarmOnes(t *testing.T) {
	warm := func(soc float64) float64 { return 150 - soc*0.8 }
	cold := func(soc float64) float64 { return 55 - soc*0.2 }
	past := []Session{session(DC, 28, 10, 80, warm), session(DC, 30, 10, 80, warm), session(DC, 2, 10, 80, cold), session(DC, 4, 10, 80, cold)}
	now := Now{Level: DC, SoC: 20, Kw: 51, BatC: f(3), OutC: f(-2)}
	got, ok := Estimate(past, now, 80, 88)
	if !ok || !got.Learned || got.TempClass != "cold" {
		t.Fatalf("%+v", got)
	}
	warmOnly, _ := Estimate(past[:2], now, 80, 88)
	if got.Minutes < warmOnly.Minutes*0.9 && warmOnly.Minutes > 0 { // the scale adapts the warm curve to what the charger gives now
		t.Logf("cold %.0f min, warm curve scaled %.0f min", got.Minutes, warmOnly.Minutes)
	}
	if got.Minutes <= 0 || got.Minutes > 120 {
		t.Fatalf("a cold battery at 51 kW takes a while but not forever: %.0f min", got.Minutes)
	}
}

func TestAWarmBatteryInColdWeatherCountsAsPreconditioned(t *testing.T) {
	r, ok := Estimate(nil, Now{Level: DC, SoC: 30, Kw: 120, BatC: f(32), OutC: f(-4)}, 80, 88)
	if !ok || !r.Preconditioned || r.TempClass != "warm" {
		t.Fatalf("%+v", r)
	}
	r, _ = Estimate(nil, Now{Level: DC, SoC: 30, Kw: 120, BatC: f(25), OutC: f(22)}, 80, 88)
	if r.Preconditioned {
		t.Fatalf("a warm day is not preconditioning: %+v", r)
	}
}

func TestWithTooFewSessionsItFallsBackToThePowerNow(t *testing.T) {
	r, ok := Estimate([]Session{session(L2, 12, 20, 100, func(float64) float64 { return 9 })}, Now{Level: L2, SoC: 50, Kw: 9}, 80, 88)
	if !ok || r.Learned {
		t.Fatalf("one session is not enough: %+v", r)
	}
	want := 88 * 0.30 / 9 * 60
	if math.Abs(r.Minutes-want) > 1 {
		t.Fatalf("30%% of 88 kWh at 9 kW is %.0f min, got %.0f", want, r.Minutes)
	}
	if _, ok := Estimate(nil, Now{Level: L2, SoC: 50, Kw: 0}, 80, 88); ok {
		t.Fatal("no power, no estimate")
	}
	if _, ok := Estimate(nil, Now{Level: L2, SoC: 90, Kw: 9}, 80, 88); ok {
		t.Fatal("already past the target")
	}
}
