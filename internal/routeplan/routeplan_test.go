package routeplan

import "testing"

func legs(km ...float64) []Leg {
	var out []Leg
	for _, k := range km {
		out = append(out, Leg{From: "A", To: "B", Km: k, Minutes: k, KwhPerKm: 0.2})
	}
	return out
}

func TestAShortRouteNeedsNoChargeAndTheBatteryFallsLegByLeg(t *testing.T) {
	// 90 kWh battery; 0.2 kWh/km: 50 km is 10 kWh, about 11.1% of the battery.
	o := Plan(Input{Legs: legs(50, 50), StartPct: 80, ReservePct: 20, TargetPct: 90, CapacityKwh: 90, ChargeKw: 100})
	if len(o.Legs) != 2 || o.Legs[0].Charge != nil || o.Legs[1].Charge != nil {
		t.Fatalf("%+v", o)
	}
	if o.Legs[0].ArrivePct < 68.8 || o.Legs[0].ArrivePct > 68.9 || o.ArrivePct < 57.6 || o.ArrivePct > 57.8 || o.TotalKm != 100 || o.TotalKwh != 20 {
		t.Fatalf("%+v", o)
	}
}

func TestALegThatWouldGoBelowTheReserveGetsAChargeBeforeItCoveringTheNextLegsToo(t *testing.T) {
	// Each 200 km leg is 40 kWh, 44.4%. Starting at 50% the first needs a charge: 20 + 44.4 and the next fits inside 90%? 20+44.4+44.4 > 90, so only this leg.
	o := Plan(Input{Legs: legs(200, 200), StartPct: 50, ReservePct: 20, TargetPct: 90, CapacityKwh: 90, ChargeKw: 90})
	c := o.Legs[0].Charge
	if c == nil || c.FromPct != 50 || c.ToPct < 66 || c.ToPct > 66.5 || c.Minutes < 9.5 || c.Minutes > 10.5 {
		t.Fatalf("a charge before the first leg: %+v", o.Legs[0])
	}
	if o.Legs[1].Charge == nil || o.Legs[1].Charge.ToPct <= o.Legs[1].Charge.FromPct {
		t.Fatalf("and another before the second: %+v", o.Legs[1])
	}
	if o.ChargingMinutes <= 0 || len(o.Warnings) != 0 {
		t.Fatalf("%+v", o)
	}
}

func TestChargingCoversTheNextShortLegsWhileTheyFitUnderTheTarget(t *testing.T) {
	o := Plan(Input{Legs: legs(200, 30, 30), StartPct: 50, ReservePct: 20, TargetPct: 90, CapacityKwh: 90, ChargeKw: 100})
	c := o.Legs[0].Charge
	if c == nil || c.ToPct < 77 { // 44.4 + 6.7 + 6.7 + reserve 20 + margin 2 = about 80
		t.Fatalf("%+v", o.Legs[0])
	}
	if o.Legs[1].Charge != nil || o.Legs[2].Charge != nil {
		t.Fatalf("no more charging needed: %+v", o.Legs)
	}
}

func TestALegTooLongForOneChargeIsFlagged(t *testing.T) {
	o := Plan(Input{Legs: legs(500), StartPct: 90, ReservePct: 20, TargetPct: 90, CapacityKwh: 90, ChargeKw: 100})
	if !o.Legs[0].Short || len(o.Warnings) != 1 {
		t.Fatalf("%+v", o)
	}
}

func TestNoCapacityGivesAnEmptyPlan(t *testing.T) {
	if o := Plan(Input{Legs: legs(10)}); len(o.Legs) != 0 {
		t.Fatalf("%+v", o)
	}
}
