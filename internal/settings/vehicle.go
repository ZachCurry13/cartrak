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
		Help: "CarTrak works out electric, hybrid or gas from Ford's data. If it is wrong, choose the type here: it decides which cards and health sections show."})
	validators[PowertrainKey] = func(val string) error {
		for _, o := range powertrainChoices {
			if val == o.Value {
				return nil
			}
		}
		return fmt.Errorf("choose one of the listed vehicle types")
	}
}
