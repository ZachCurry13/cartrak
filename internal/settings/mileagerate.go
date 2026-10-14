package settings

import (
	"fmt"
	"strconv"
)

// MileageRateKey is what one mile of business driving is worth as a deduction, in the currency set above.
const MileageRateKey = "CARTRAK_MILEAGE_RATE"

func init() {
	Defs = append(Defs, Def{Key: MileageRateKey, Label: "Mileage deduction per mile", Placeholder: "0.70",
		Help: "Optional. Your tax authority's rate for business driving, per mile. The Mileage log and Year in review then show an estimated deduction for work trips. It is an estimate for your own planning, not tax advice."})
	validators[MileageRateKey] = func(v string) error {
		if v == "" {
			return nil
		}
		if n, err := strconv.ParseFloat(v, 64); err != nil || n <= 0 || n > 100 {
			return fmt.Errorf("enter an amount per mile such as 0.70")
		}
		return nil
	}
}

// MileageRate is the deduction per mile, or nil when none is set.
func (s *Settings) MileageRate() *float64 {
	if n, err := strconv.ParseFloat(s.Value(MileageRateKey), 64); err == nil && n > 0 {
		return &n
	}
	return nil
}
