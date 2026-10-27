package savings

import "testing"

func TestEfficiencyIsTheLastThirtyDaysBeforeTheCharge(t *testing.T) {
	d := int64(day)
	e := NewEfficiency([]Drive{
		{TS: 10 * d, Km: 100, Kwh: 25}, // 4 km/kWh, long ago
		{TS: 70 * d, Km: 80, Kwh: 20},  // 4 km/kWh
		{TS: 80 * d, Km: 60, Kwh: 20},  // 3 km/kWh
		{TS: 90 * d, Km: 90, Kwh: 30},  // 3 km/kWh, after the charge below
	})
	if got := e.KmPerKwh(85 * d); !near(got, 140.0/40) { // 70d and 80d fall in the window
		t.Fatalf("window: %.3f", got)
	}
	if got := e.KmPerKwh(30 * d); got != 0 && !near(got, 4) { // thin window: widens to everything before
		t.Fatalf("widened: %.3f", got)
	}
	if got := e.KmPerKwh(5 * d); !near(got, 330.0/95) { // nothing before: all drives
		t.Fatalf("first charge uses every drive: %.3f", got)
	}
	if got := NewEfficiency(nil).KmPerKwh(d); got != 0 {
		t.Fatalf("no trips, no figure: %.3f", got)
	}
	if got := NewEfficiency([]Drive{{TS: 1, Km: 100, Kwh: 0}, {TS: 2, Km: 0, Kwh: 5}}).KmPerKwh(d); got != 0 {
		t.Fatalf("drives without energy or distance are ignored: %.3f", got)
	}
}

func TestSavingsComeFromTheEnergyChargedNotTheDistanceDriven(t *testing.T) {
	// 37 kWh at 0.42 each, at 4 km/kWh: 148 km, which a 8 L/100 km car burns 11.84 L for, at 1.00 per liter.
	km := 37 * 4.0
	liters, gasCost := Gas(km, 0.08, 1.0)
	r := Compute(Input{Km: 999, Sessions: 1, Priced: 1, ChargingCost: 37 * 0.42, WallKwh: 40, GridKgPerKwh: 0.37,
		Equiv: &Equiv{Km: km, Liters: liters, GasCost: gasCost, Paid: 37 * 0.42, Priced: 1, Sources: []string{"Minnesota average"}}})
	if r.Saved == nil || !near(*r.Saved, 11.84-15.54) {
		t.Fatalf("saved %v", r.Saved)
	}
	if r.EquivKm == nil || !near(*r.EquivKm, 148) || r.GasCost == nil || !near(*r.GasCost, 11.84) {
		t.Fatalf("the distance driven (999) must not matter: %+v", r)
	}
	if r.CO2AvoidedKg == nil || !near(*r.CO2AvoidedKg, 11.84*kgCO2PerLiter-40*0.37) {
		t.Fatalf("co2 %v", r.CO2AvoidedKg)
	}
}

func TestUnpricedChargesAreNotCountedAsFree(t *testing.T) {
	r := Compute(Input{Sessions: 2, Priced: 0, Equiv: &Equiv{Km: 100, Liters: 8, Skipped: 1}})
	if r.Saved != nil || r.GasCost != nil || r.Skipped != 1 || r.GasLiters == nil {
		t.Fatalf("no price, no saving: %+v", r)
	}
}
