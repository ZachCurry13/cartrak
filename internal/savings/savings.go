// Package savings works out what driving electric has cost against a gas car, and the CO2 it avoided.
package savings

// kgCO2PerLitre is the CO2 from burning a litre of gas (8.887 kg per US gallon).
const kgCO2PerLitre = 2.348

// Input is one period: the distance driven, what charging cost, and the gas car to compare with.
type Input struct {
	Km            float64
	Sessions      int     // charging sessions in the period
	Priced        int     // of them, how many have a cost
	ChargingCost  float64 // of the priced sessions
	WallKwh       float64 // energy taken from the grid, for the CO2
	LitresPerKm   float64 // the gas car; zero when not set
	PricePerLitre float64
	GridKgPerKwh  float64
}

// Result is what the period comes to. Pointers are nil when they cannot be told.
type Result struct {
	Km           float64  `json:"km"`
	ChargingCost float64  `json:"chargingCost"`
	Partial      bool     `json:"partial"`   // some sessions have no price, so the cost is a floor
	CostPerKm    *float64 `json:"costPerKm"` // charging cost per distance driven; only when every session has a price
	GasCost      *float64 `json:"gasCost"`   // what the same distance would have cost in the gas car
	Saved        *float64 `json:"saved"`     // gas cost minus charging cost
	GasLitres    *float64 `json:"gasLitres"`
	CO2AvoidedKg *float64 `json:"co2AvoidedKg"`
}

// Compute fills in a Result. The saving needs at least one priced session and a gas car; the CO2 needs only the gas car.
func Compute(in Input) Result {
	r := Result{Km: in.Km, ChargingCost: in.ChargingCost, Partial: in.Sessions > in.Priced}
	if in.Km > 0 && in.Priced > 0 && in.Priced == in.Sessions {
		c := in.ChargingCost / in.Km
		r.CostPerKm = &c
	}
	if in.Km <= 0 || in.LitresPerKm <= 0 {
		return r
	}
	litres := in.Km * in.LitresPerKm
	r.GasLitres = &litres
	if in.PricePerLitre > 0 {
		pc := litres * in.PricePerLitre
		r.GasCost = &pc
		if in.Priced > 0 {
			s := pc - in.ChargingCost
			r.Saved = &s
		}
	}
	grid := in.GridKgPerKwh
	if grid <= 0 {
		grid = 0.37
	}
	co2 := litres*kgCO2PerLitre - in.WallKwh*grid
	r.CO2AvoidedKg = &co2
	return r
}
