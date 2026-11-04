package alerts

import "testing"

func TestADistanceIsShownInOneUnit(t *testing.T) {
	e := &Engine{}
	if got := e.distance(45, 1); got != "28.0 mi" {
		t.Errorf("miles by default: %q", got)
	}
	e.Metric = func() bool { return true }
	if got := e.distance(45, 1); got != "45.0 km" {
		t.Errorf("metric: %q", got)
	}
}
