package alerts

import (
	"testing"

	"github.com/ZachCurry13/cartrak/internal/ford"
)

func tank(pct float64) func(*ford.State) {
	return func(s *ford.State) { s.FuelPct = &pct; s.SoCPct = nil }
}

func TestAJumpInTheFuelGaugeIsAFillUpKeptOnceToComplete(t *testing.T) {
	e, r, s := setup(t)
	odo := 80000.0
	e.Handle(reading(0, tank(22)))
	e.Handle(reading(10, tank(21))) // slosh, and the car using a little
	if len(r.got) != 0 {
		t.Fatalf("a small change is not a fill-up: %s", r.kinds())
	}
	e.Handle(reading(20, func(st *ford.State) { tank(60)(st); st.OdometerKm = &odo }))
	e.Handle(reading(30, tank(97))) // pumping on: the same fill-up, higher
	if r.kinds() != "fill_up" {
		t.Fatalf("one alert for the fill-up: %s", r.kinds())
	}
	list, _ := s.ListRefuels()
	if len(list) != 1 || list[0].StartPct != 21 || list[0].EndPct != 97 || list[0].Status != "pending" || list[0].OdoKm == nil {
		t.Fatalf("kept, with the levels before and after: %+v", list)
	}
	e.Handle(reading(300, tank(30)))
	e.Handle(reading(320, tank(95))) // a later one is another fill-up
	if r.kinds() != "fill_up,fill_up" {
		t.Fatalf("a second fill-up: %s", r.kinds())
	}
}

func TestADrivingCarsGaugeIsNotAFillUp(t *testing.T) {
	e, r, _ := setup(t)
	e.Handle(reading(0, tank(30)))
	e.Handle(reading(5, func(s *ford.State) { tank(80)(s); sp := 60.0; s.SpeedKph = &sp }))
	if len(r.got) != 0 {
		t.Fatalf("not while moving: %s", r.kinds())
	}
}

func TestLowFuelIsToldOnceUntilFilled(t *testing.T) {
	e, r, _ := setup(t)
	e.Handle(reading(0, tank(40)))
	e.Handle(reading(10, tank(14)))
	e.Handle(reading(20, tank(13)))
	if r.kinds() != "low_fuel" {
		t.Fatalf("once: %s", r.kinds())
	}
	e.Handle(reading(30, tank(90))) // filled: low fuel re-arms, and the fill-up is noticed
	e.Handle(reading(40, tank(12)))
	if r.kinds() != "low_fuel,fill_up,low_fuel" {
		t.Fatalf("again after a fill-up: %s", r.kinds())
	}
}
