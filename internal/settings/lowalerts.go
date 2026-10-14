package settings

import (
	"fmt"
	"strconv"
)

// Two optional alerts with a number to choose: a low battery level and a monthly charging budget.
// Each device still turns the alert itself on under More, Notifications.
const (
	LowBatteryKey   = "CARTRAK_LOW_BATTERY"
	ChargeBudgetKey = "CARTRAK_CHARGE_BUDGET"
)

var lowBatteryLevels = []Option{{"", "Off"}, {"10", "10%"}, {"15", "15%"}, {"20", "20%"}, {"25", "25%"}, {"30", "30%"}}

func init() {
	Defs = append(Defs,
		Def{Key: LowBatteryKey, Label: "Low battery alert level", Options: lowBatteryLevels,
			Help: "Devices that turned on the Low battery alert are told once when the battery falls to this level while the car is not charging. It resets after the battery is five points higher or the car charges."},
		Def{Key: ChargeBudgetKey, Label: "Monthly charging budget", Placeholder: "for example 60",
			Help: "Devices that turned on the Charging budget alert are told once a month when the month's charging cost passes this amount, in the currency set under prices. Needs your price per kWh. Leave empty for no budget."})
	validators[LowBatteryKey] = func(v string) error {
		for _, o := range lowBatteryLevels {
			if o.Value == v {
				return nil
			}
		}
		return fmt.Errorf("choose one of the levels")
	}
	validators[ChargeBudgetKey] = func(v string) error {
		if v == "" {
			return nil
		}
		if n, err := strconv.ParseFloat(v, 64); err != nil || n <= 0 || n > 100000 {
			return fmt.Errorf("enter an amount such as 60")
		}
		return nil
	}
}

// LowBatteryLevel is the percent at which to warn, or 0 for off.
func (s *Settings) LowBatteryLevel() float64 {
	n, _ := strconv.ParseFloat(s.Value(LowBatteryKey), 64)
	return n
}

// ChargeBudget is the monthly charging budget, or 0 for none.
func (s *Settings) ChargeBudget() float64 {
	n, _ := strconv.ParseFloat(s.Value(ChargeBudgetKey), 64)
	return n
}
