package savings

// withEquiv fills in the gas side of a Result from charge-by-charge figures.
func withEquiv(r Result, in Input) Result {
	e := in.Equiv
	r.Skipped, r.Sources = e.Skipped, e.Sources
	if e.Km <= 0 {
		return r
	}
	km, liters := e.Km, e.Liters
	r.EquivKm, r.GasLiters = &km, &liters
	if e.Priced > 0 {
		gc, saved := e.GasCost, e.GasCost-e.Paid
		r.GasCost, r.Saved = &gc, &saved
	}
	grid := in.GridKgPerKwh
	if grid <= 0 {
		grid = 0.37
	}
	co2 := liters*kgCO2PerLiter - in.WallKwh*grid
	r.CO2AvoidedKg = &co2
	return r
}

// Gas is the gas car's fuel and cost for the distance some energy would go.
func Gas(km, litersPerKm, pricePerLiter float64) (liters, cost float64) {
	liters = km * litersPerKm
	return liters, liters * pricePerLiter
}
