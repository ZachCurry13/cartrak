package trips

import (
	"math"
	"testing"

	"github.com/ZachCurry13/cartrak/internal/store"
)

func TestTheOdometerCorrectionScalesOdometerDistancesOnly(t *testing.T) {
	a, b := 1000.0, 1100.0
	tr := &store.Trip{StartOdo: &a, EndOdo: &b}
	if got := (&Detector{}).distance(tr); got != 100 {
		t.Fatalf("no correction: %.2f", got)
	}
	d := &Detector{OdoFactor: func() float64 { return 0.973 }}
	if got := d.distance(tr); math.Abs(got-97.3) > 1e-9 {
		t.Fatalf("100 km on the odometer with a correction of 0.973: %.3f", got)
	}
	for _, bad := range []float64{0, -1} { // a missing or nonsense value never zeroes a trip
		if got := (&Detector{OdoFactor: func() float64 { return bad }}).distance(tr); got != 100 {
			t.Fatalf("factor %.0f is ignored: %.2f", bad, got)
		}
	}
}
