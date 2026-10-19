package server

import "testing"

func TestTheRegenCounterIsTrustedOnlyWhenMostTripsAgree(t *testing.T) {
	if !regenRatiosAgree([]float64{1.0, 0.95, 1.1, 1.02}) {
		t.Fatal("close agreement is trusted")
	}
	if regenRatiosAgree([]float64{1.0, 1.0}) {
		t.Fatal("two trips are not enough")
	}
	if regenRatiosAgree([]float64{10, 9.5, 11, 10.2}) {
		t.Fatal("a counter ten times too big is not trusted")
	}
	if regenRatiosAgree([]float64{1.0, 1.0, 1.0, 3.0, 0.2}) {
		t.Fatal("too many disagreeing trips")
	}
}
