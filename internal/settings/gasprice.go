package settings

import (
	"fmt"
	"strconv"
	"strings"
)

// Live gas prices: the free EIA key, and the owner's own adjustments for places that cost more or less than the area average.
const (
	EIAKeyKey    = "CARTRAK_EIA_KEY"
	GasAdjustKey = "CARTRAK_GAS_ADJUST"
)

// AreaAdjust is a percentage added to the gas price in one city or state ("Springfield", "+15").
type AreaAdjust struct {
	Area string
	Pct  float64
}

func init() {
	Defs = append(Defs,
		Def{Key: EIAKeyKey, Label: "EIA key (live gas prices)", Secret: true,
			Help: "Optional. Lets CarTrak look up the weekly average gas price where each charge happened, and compare it with what you paid. To get a free key: open eia.gov/opendata, choose Register, fill in the short form, and paste the key EIA gives you here. Stored on the server only. Without it, your gas price above is used."},
		Def{Key: GasAdjustKey, Label: "Gas price by place (optional)", Placeholder: "Springfield +15%; Hillcrest -5%",
			Help: "Places that cost more or less than the area average. A city or a two-letter state, then a percentage, separated by semicolons."})
	validators[GasAdjustKey] = func(v string) error {
		if _, err := parseAdjusts(v); err != nil {
			return err
		}
		return nil
	}
}

func parseAdjusts(v string) ([]AreaAdjust, error) {
	var out []AreaAdjust
	for _, part := range strings.Split(v, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		i := strings.LastIndexAny(part, " =:")
		if i < 1 {
			return nil, fmt.Errorf("write each as a place and a percentage, such as Springfield +15%%")
		}
		n, err := strconv.ParseFloat(strings.TrimSuffix(strings.TrimSpace(part[i+1:]), "%"), 64)
		if err != nil || n < -90 || n > 200 {
			return nil, fmt.Errorf("%q needs a percentage between -90 and 200", part)
		}
		out = append(out, AreaAdjust{Area: strings.TrimSpace(strings.TrimRight(part[:i], " =:")), Pct: n})
	}
	return out, nil
}

// GasAdjusts is the owner's per-place gas price adjustments.
func (s *Settings) GasAdjusts() []AreaAdjust {
	a, _ := parseAdjusts(s.Value(GasAdjustKey))
	return a
}

// EIAKey is the key for live gas prices, "" when not set.
func (s *Settings) EIAKey() string { return strings.TrimSpace(s.Value(EIAKeyKey)) }

// LiveGasUnits reports whether live prices apply: US units, since the EIA gives dollars per gallon.
func (s *Settings) LiveGasUnits() bool { return s.Value(FuelUnitsKey) != "metric" }
