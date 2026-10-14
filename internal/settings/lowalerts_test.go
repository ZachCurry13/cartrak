package settings

import "testing"

func TestAlertLevelsBudgetAndMileageRateAreValidatedAndRead(t *testing.T) {
	for _, c := range []struct {
		key, val string
		ok       bool
	}{
		{LowBatteryKey, "20", true}, {LowBatteryKey, "", true}, {LowBatteryKey, "17", false},
		{ChargeBudgetKey, "60", true}, {ChargeBudgetKey, "", true}, {ChargeBudgetKey, "-5", false}, {ChargeBudgetKey, "lots", false},
		{MileageRateKey, "0.70", true}, {MileageRateKey, "", true}, {MileageRateKey, "0", false}, {MileageRateKey, "abc", false},
	} {
		err := validators[c.key](c.val)
		if (err == nil) != c.ok {
			t.Errorf("%s=%q: err=%v, want ok=%v", c.key, c.val, err, c.ok)
		}
	}
}
