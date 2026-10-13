package maint

import (
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

func f(v float64) *float64 { return &v }
func i(v int) *int         { return &v }

func TestAReminderIsOkSoonOrOverdueByDistanceOrByTime(t *testing.T) {
	now := time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC).Unix()
	day := int64(86400)
	km := &store.Maintenance{Name: "Tire rotation", EveryKm: f(10000), LastOdoKm: f(20000), LastTS: now - 30*day}
	for _, c := range []struct {
		odo   float64
		state State
	}{{25000, OK}, {29600, Soon}, {30000, Overdue}, {31000, Overdue}} {
		if got := Check(km, f(c.odo), now); got.State != c.state {
			t.Errorf("odometer %.0f: %s, want %s", c.odo, got.State, c.state)
		}
	}
	if got := Check(km, f(29000), now); got.KmLeft == nil || *got.KmLeft != 1000 {
		t.Fatalf("%+v", got.KmLeft)
	}
	cabin := &store.Maintenance{Name: "Cabin filter", EveryDays: i(730), LastTS: now - 740*day}
	if got := Check(cabin, nil, now); got.State != Overdue || got.DaysLeft == nil || *got.DaysLeft >= 0 {
		t.Fatalf("two years and ten days: %+v", got)
	}
	if got := Check(&store.Maintenance{EveryDays: i(365), LastTS: now - 345*day}, nil, now); got.State != Soon {
		t.Fatalf("20 days left is soon: %+v", got)
	}
	if got := Check(&store.Maintenance{EveryKm: f(10000), LastOdoKm: f(1)}, nil, now); got.State != OK || got.KmLeft != nil {
		t.Fatalf("with no odometer there is nothing to compare: %+v", got)
	}
	both := &store.Maintenance{EveryKm: f(10000), LastOdoKm: f(0), EveryDays: i(180), LastTS: now - 200*day}
	if got := Check(both, f(1000), now); got.State != Overdue {
		t.Fatalf("either interval can make it due: %+v", got)
	}
}
