package push

import "testing"

func TestEverydayAlertsAreQuietAndWarningsKeepTheirSound(t *testing.T) {
	for _, k := range []string{TripStart, TripEnd, PlugIn, PlugOut, ArrivePlace, LeavePlace, Achievement} {
		if !Quiet(k) {
			t.Errorf("%s should be quiet", k)
		}
	}
	for _, k := range []string{AlarmSounding, LowTire, Low12V, LowBattery, CarFault, FordTrouble, ChargeComplete, Unlocked, LeftOpen, LowDisk, MaintenanceDue, PlugReminder} {
		if Quiet(k) {
			t.Errorf("%s needs a person's attention and should keep its sound", k)
		}
	}
	for k := range quiet {
		if !ValidKind(k) {
			t.Errorf("%s is not an alert kind", k)
		}
	}
}
