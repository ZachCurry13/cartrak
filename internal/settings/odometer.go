package settings

import (
	"fmt"
	"strconv"
)

// OdometerFactorKey scales the distance worked out from the car's odometer. A GPS mileage app reads a little short and an odometer a little
// long; a factor such as 0.97 makes CarTrak's distances agree with the one the owner trusts.
const OdometerFactorKey = "CARTRAK_ODOMETER_FACTOR"

func init() {
	Defs = append(Defs, Def{Key: OdometerFactorKey, Label: "Odometer correction", Placeholder: "for example 0.973",
		Help: "Multiplies every distance worked out from the car's odometer. GPS apps read a few percent shorter than an odometer: enter 0.97 to match one. Easier: open a trip, \"Does the distance look wrong?\", and type what the other app said. New trips use it; Fix a day applies it to older ones. Empty: 1."})
	validators[OdometerFactorKey] = func(v string) error {
		if v == "" {
			return nil
		}
		if n, err := strconv.ParseFloat(v, 64); err != nil || n < 0.8 || n > 1.2 {
			return fmt.Errorf("enter a number between 0.80 and 1.20, such as 0.973")
		}
		return nil
	}
}

// OdometerFactor is the correction for odometer distances, 1 when not set.
func (s *Settings) OdometerFactor() float64 {
	if n, err := strconv.ParseFloat(s.Value(OdometerFactorKey), 64); err == nil && n >= 0.8 && n <= 1.2 {
		return n
	}
	return 1
}
