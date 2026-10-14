package settings

import (
	"fmt"
	"strconv"
	"strings"
)

// What charging costs, for the cost figures on the Charging page. Blank means "not set".
const (
	PriceHomeKey   = "CARTRAK_PRICE_HOME"
	PricePublicKey = "CARTRAK_PRICE_PUBLIC"
	CurrencyKey    = "CARTRAK_CURRENCY"
	ChargeEffKey   = "CARTRAK_CHARGE_EFFICIENCY"
)

func init() {
	Defs = append(Defs,
		Def{Key: PriceHomeKey, Label: "Price per kWh at home", Placeholder: "0.14",
			Help: "What you pay your electricity company per kWh, for example 0.14. Charging at home is priced with this."},
		Def{Key: PricePublicKey, Label: "Price per kWh when charging away from home", Placeholder: "0.45",
			Help: "An average for public chargers, for example 0.45. Leave empty to use the home price for everything."},
		Def{Key: CurrencyKey, Label: "Currency symbol", Options: []Option{{"", "$"}, {"€", "€"}, {"£", "£"}, {"kr", "kr"}, {"A$", "A$"}, {"C$", "C$"}},
			Help: "Shown in front of the cost figures."})
	Defs = append(Defs, Def{Key: ChargeEffKey, Label: "Slow-charging efficiency", Options: effOptions,
		Help: "How much of the energy from a home charger reaches the battery. Used for cost estimates until at least three sessions have the charger's own number entered, then your measured average is used instead."})
	validators[ChargeEffKey] = func(v string) error {
		for _, o := range effOptions {
			if o.Value == v {
				return nil
			}
		}
		return fmt.Errorf("choose one of the values")
	}
	price := func(v string) error {
		if v == "" {
			return nil
		}
		n, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err != nil || n < 0 || n > 20 {
			return fmt.Errorf("enter a price per kWh such as 0.14")
		}
		return nil
	}
	validators[PriceHomeKey], validators[PricePublicKey] = price, price
	validators[CurrencyKey] = func(v string) error {
		if len(v) > 4 {
			return fmt.Errorf("use a short symbol such as $")
		}
		return nil
	}
}

var effOptions = []Option{{"", "90% (default)"}, {"80", "80%"}, {"84", "84%"}, {"86", "86%"}, {"88", "88%"}, {"92", "92%"}, {"95", "95%"}}

// ChargeEfficiency is the share (0.8 to 0.95) of wall energy assumed to reach the battery.
func (s *Settings) ChargeEfficiency() float64 {
	if n, err := strconv.Atoi(s.Value(ChargeEffKey)); err == nil && n >= 50 && n <= 100 {
		return float64(n) / 100
	}
	return 0.9
}

func (s *Settings) price(key string) *float64 {
	if n, err := strconv.ParseFloat(strings.TrimSpace(s.Value(key)), 64); err == nil && n >= 0 && s.Value(key) != "" {
		return &n
	}
	return nil
}

// PriceHome is the price per kWh at home, nil when not set.
func (s *Settings) PriceHome() *float64 { return s.price(PriceHomeKey) }

// PricePublic is the price per kWh away from home. When it is not set the home price is used.
func (s *Settings) PricePublic() *float64 {
	if p := s.price(PricePublicKey); p != nil {
		return p
	}
	return s.PriceHome()
}

// Currency is the symbol shown in front of costs.
func (s *Settings) Currency() string {
	if v := s.Value(CurrencyKey); v != "" {
		return v
	}
	return "$"
}
