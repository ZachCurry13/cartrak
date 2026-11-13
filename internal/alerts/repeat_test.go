package alerts

import (
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
)

// A door left open as the car went to sleep is told about once, not again every twelve hours while the car repeats that report.
func TestTheSameOpenDoorReportIsNotRepeatedDaysLater(t *testing.T) {
	e, r, _ := setup(t)
	p := "telemetry.metrics.doorStatus"
	stamp := t0.Add(-5 * time.Minute).UTC().Format(time.RFC3339)
	open := func(s *ford.State) {
		s.Extra[p+"[0].value"] = "AJAR"
		s.Extra[p+"[0].vehicleDoor"] = "UNSPECIFIED_FRONT"
		s.Extra[p+"[0].vehicleSide"] = "DRIVER"
		s.Extra[p+"[0].updateTime"] = stamp
	}
	for i := 0; i < 5; i++ {
		e.Handle(reading(i*4, open))
	}
	r2 := &rec{}
	e2 := &Engine{S: e.S, N: r2} // a restart a day later, the car still asleep with the same report
	for i := 0; i < 5; i++ {
		e2.Handle(reading(24*60+i*4, open))
	}
	e3 := &Engine{S: e.S, N: r2}
	for i := 0; i < 5; i++ {
		e3.Handle(reading(38*60+i*4, open))
	}
	if r.kinds() != "left_open" || len(r2.got) != 0 {
		t.Fatalf("once: %s, then %s", r.kinds(), r2.kinds())
	}
}

// A car left unlocked is told about once for that parked spell, even when it reports again (with a newer time) after a restart; once
// it has been locked, being left unlocked is news again.
func TestOneUnlockedSpellIsToldOnceUntilTheCarIsLocked(t *testing.T) {
	e, r, _ := setup(t)
	unlockedAt := func(min int) func(*ford.State) {
		return func(s *ford.State) {
			f := false
			s.Locked = &f
			s.Extra["telemetry.metrics.doorLockStatus[0].value"] = "UNLOCKED"
			s.Extra["telemetry.metrics.doorLockStatus[0].vehicleDoor"] = "ALL_DOORS"
			s.Extra["telemetry.metrics.doorLockStatus[0].updateTime"] = t0.Add(time.Duration(min) * time.Minute).UTC().Format(time.RFC3339)
		}
	}
	for i := 0; i < 5; i++ {
		e.Handle(reading(i*4, unlockedAt(0)))
	}
	r2 := &rec{}
	e2 := &Engine{S: e.S, N: r2}
	for i := 0; i < 5; i++ { // restarted; the car woke at 6 pm and still says unlocked
		e2.Handle(reading(18*60+i*4, unlockedAt(18*60)))
	}
	if r.kinds() != "unlocked" || len(r2.got) != 0 {
		t.Fatalf("once: %s, then %s", r.kinds(), r2.kinds())
	}
	e2.Handle(reading(20*60, func(s *ford.State) { l := true; s.Locked = &l }))
	for i := 0; i < 5; i++ {
		e2.Handle(reading(22*60+i*4, unlockedAt(22*60)))
	}
	if r2.kinds() != "unlocked" {
		t.Fatalf("locked, then left unlocked again, is news: %s", r2.kinds())
	}
}
