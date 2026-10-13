package settings

import "fmt"

// PowertrainKey overrides the type CarTrak works out for the car; empty means auto-detect.
const PowertrainKey = "CARTRAK_POWERTRAIN"

var powertrainChoices = []Option{
	{"", "Auto-detect from what the car reports"},
	{"ev", "Electric (battery only)"},
	{"phev", "Plug-in hybrid"},
	{"hev", "Hybrid (no plug)"},
	{"gas", "Gas or diesel"},
}

func init() {
	Defs = append(Defs, Def{Key: PowertrainKey, Label: "Vehicle type", Options: powertrainChoices,
		Help: "CarTrak works out whether the car is electric, hybrid or gas from the data Ford sends. If it guesses wrong, choose the type here. It decides which cards and Vehicle health sections appear."})
	validators[PowertrainKey] = func(val string) error {
		for _, o := range powertrainChoices {
			if val == o.Value {
				return nil
			}
		}
		return fmt.Errorf("choose one of the listed vehicle types")
	}
}
