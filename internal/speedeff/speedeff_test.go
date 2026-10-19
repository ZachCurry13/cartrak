package speedeff

import (
	"math"
	"testing"
)

// drive makes readings every 30 s at a constant speed that use perKm kWh per kilometre.
func drive(startTS int64, odo, kwh, kph, perKm float64, n int) []Reading {
	var out []Reading
	for i := 0; i <= n; i++ {
		d := kph / 3600 * 30 * float64(i)
		out = append(out, Reading{TS: startTS + int64(i)*30, Odo: odo + d, Kwh: kwh - d*perKm, Speed: kph})
	}
	return out
}

func TestFasterBandsUseMoreEnergyPerKilometre(t *testing.T) {
	rs := append(drive(0, 1000, 80, 70, 0.14, 120), drive(10000, 1100, 60, 112, 0.22, 120)...)
	got := Build(rs)
	if len(got) != 2 {
		t.Fatalf("two bands: %+v", got)
	}
	if got[0].LoKph != 50 || math.Abs(got[0].KmPerKwh-1/0.14) > 0.1 {
		t.Fatalf("70 km/h: %+v", got[0])
	}
	if got[1].LoKph != 100 || math.Abs(got[1].KmPerKwh-1/0.22) > 0.1 {
		t.Fatalf("112 km/h: %+v", got[1])
	}
	if got[1].KmPerKwh >= got[0].KmPerKwh {
		t.Fatal("going faster must cost more")
	}
}

func TestStopsGapsChargesAndTooLittleDrivingAreLeftOut(t *testing.T) {
	rs := drive(0, 1000, 80, 70, 0.14, 4) // only about 2.3 km: too little to show
	if got := Build(rs); len(got) != 0 {
		t.Fatalf("not enough driving: %+v", got)
	}
	long := drive(0, 1000, 80, 70, 0.14, 120)
	long = append(long, Reading{TS: 100000, Odo: long[len(long)-1].Odo + 1, Kwh: 20, Speed: 70}) // a gap of hours: skipped
	long = append(long, Reading{TS: 100030, Odo: long[len(long)-1].Odo + 1, Kwh: 60, Speed: 70}) // energy jumped up: a charge
	got := Build(long)
	if len(got) != 1 || math.Abs(got[0].KmPerKwh-1/0.14) > 0.1 {
		t.Fatalf("only real driving counts: %+v", got)
	}
}
