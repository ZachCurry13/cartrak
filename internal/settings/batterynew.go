package settings

import (
	"fmt"
	"strconv"
)

// BatteryNewKey is what the battery could hold when new, so Battery health can say how much of that is left.
const BatteryNewKey = "CARTRAK_BATTERY_NEW_KWH"

func init() {
	Defs = append(Defs, Def{Key: BatteryNewKey, Label: "Battery size when new (kWh)", Placeholder: "for example 88",
		Help: "Battery health is shown as a percentage of this. Use your model's usable size (Ford does not send it). Empty: compared with the first weeks CarTrak saw."})
	validators[BatteryNewKey] = func(v string) error {
		if v == "" {
			return nil
		}
		if n, err := strconv.ParseFloat(v, 64); err != nil || n < 10 || n > 300 {
			return fmt.Errorf("enter the size in kWh, such as 88")
		}
		return nil
	}
}

// BatteryNewKwh is the battery's size when new, or 0 when not set.
func (s *Settings) BatteryNewKwh() float64 {
	if n, err := strconv.ParseFloat(s.Value(BatteryNewKey), 64); err == nil && n > 0 {
		return n
	}
	return 0
}
