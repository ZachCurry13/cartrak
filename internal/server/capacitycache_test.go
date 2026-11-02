package server

import (
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
)

// The capacity used for trip energy is the middle of the last weeks' readings, not whatever the latest one says.
func TestCapacityUsedForEnergyIgnoresOddReadings(t *testing.T) {
	_, srv := healthServer(t, fakeProv{}, false, false)
	now := time.Now()
	reads := []float64{88, 88.5, 88, 92.5, 88, 88.2, 85.6}
	for i, kwh := range reads {
		c := kwh
		if _, err := srv.S.SaveState("t", &ford.State{TS: now.Add(-time.Duration(len(reads)-i) * time.Hour), CapacityKwh: &c, Raw: []byte(`{}`)}); err != nil {
			t.Fatal(err)
		}
	}
	got := srv.capacityKwh()
	if got == nil || *got < 87.9 || *got > 88.3 {
		t.Fatalf("capacity %v, want about 88", got)
	}
}

func TestCapacityFallsBackToTheLatestReadingWhenThereAreFew(t *testing.T) {
	_, srv := healthServer(t, fakeProv{}, false, false)
	c := 90.0
	if _, err := srv.S.SaveState("t", &ford.State{TS: time.Now(), CapacityKwh: &c, Raw: []byte(`{}`)}); err != nil {
		t.Fatal(err)
	}
	if got := srv.capacityKwh(); got == nil || *got != 90 {
		t.Fatalf("capacity %v", got)
	}
}
