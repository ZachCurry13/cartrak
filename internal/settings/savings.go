package settings

import (
	"fmt"
	"strconv"
	"strings"
)

// The petrol car you would otherwise be driving, for the savings and CO2 figures. Blank means "not set", and then nothing is
// compared.
const (
	FuelUnitsKey     = "CARTRAK_FUEL_UNITS"
	PetrolPriceKey   = "CARTRAK_PETROL_PRICE"
	PetrolEconomyKey = "CARTRAK_PETROL_ECONOMY"
	GridCO2Key       = "CARTRAK_GRID_CO2"
)

const (
	litresPerGallon = 3.785411784
	mpgToLPer100Km  = 235.2145833 // L/100 km = this / mpg (US)
	defaultGridCO2  = 0.37        // kg of CO2 per kWh, roughly the US grid average
)

func init() {
	Defs = append(Defs,
		Def{Key: FuelUnitsKey, Label: "Fuel units", Options: []Option{{"", "US gallons and mpg"}, {"metric", "Litres and L per 100 km"}},
			Help: "How you give the petrol price and the economy of the petrol car you would otherwise drive."},
		Def{Key: PetrolPriceKey, Label: "Petrol price (per gallon or litre)", Placeholder: "3.50",
			Help: "What petrol costs you, per gallon or per litre to match the units above. With the economy below, CarTrak shows what you save by driving electric."},
		Def{Key: PetrolEconomyKey, Label: "Petrol car's economy (mpg or L per 100 km)", Placeholder: "28",
			Help: "How far the petrol car you would otherwise drive goes on fuel: miles per gallon, or litres per 100 km with the metric option."},
		Def{Key: GridCO2Key, Label: "CO2 per kWh of electricity (kg)", Placeholder: "0.37",
			Help: "How much CO2 your electricity makes per kWh, used for the CO2 figure. The default is about the US average; your electricity company or national grid publishes yours."})
	positive := func(max float64, hint string) func(string) error {
		return func(v string) error {
			if v == "" {
				return nil
			}
			if n, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err != nil || n <= 0 || n > max {
				return fmt.Errorf("%s", hint)
			}
			return nil
		}
	}
	validators[FuelUnitsKey] = func(v string) error {
		if v != "" && v != "metric" {
			return fmt.Errorf("choose one of the values")
		}
		return nil
	}
	validators[PetrolPriceKey] = positive(50, "enter a price such as 3.50")
	validators[PetrolEconomyKey] = positive(200, "enter a number such as 28")
	validators[GridCO2Key] = positive(2, "enter kg of CO2 per kWh, such as 0.37")
}

// PetrolCar is the petrol alternative, in units that do not depend on how it was entered. LitresPerKm and PricePerLitre are zero
// when it is not set.
type PetrolCar struct {
	LitresPerKm   float64
	PricePerLitre float64
}

// Set reports whether both the price and the economy were given.
func (p PetrolCar) Set() bool { return p.LitresPerKm > 0 && p.PricePerLitre > 0 }

// Petrol is the petrol car to compare against.
func (s *Settings) Petrol() PetrolCar {
	price, e1 := strconv.ParseFloat(strings.TrimSpace(s.Value(PetrolPriceKey)), 64)
	econ, e2 := strconv.ParseFloat(strings.TrimSpace(s.Value(PetrolEconomyKey)), 64)
	if e1 != nil || e2 != nil || price <= 0 || econ <= 0 {
		return PetrolCar{}
	}
	if s.Value(FuelUnitsKey) == "metric" {
		return PetrolCar{LitresPerKm: econ / 100, PricePerLitre: price}
	}
	return PetrolCar{LitresPerKm: mpgToLPer100Km / econ / 100, PricePerLitre: price / litresPerGallon}
}

// GridCO2 is the kg of CO2 per kWh of electricity.
func (s *Settings) GridCO2() float64 {
	if n, err := strconv.ParseFloat(strings.TrimSpace(s.Value(GridCO2Key)), 64); err == nil && n > 0 {
		return n
	}
	return defaultGridCO2
}
