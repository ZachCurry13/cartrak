package settings

import (
	"fmt"
	"strconv"
)

// The evening "plug in tonight?" reminder. Each device still chooses whether to receive it under
// More, Notifications; these two settings say when it applies.
const (
	PlugReminderTimeKey  = "CARTRAK_PLUG_REMINDER_TIME"
	PlugReminderBelowKey = "CARTRAK_PLUG_REMINDER_BELOW"
)

var reminderTimes = []Option{{"", "9 pm (default)"}, {"17:00", "5 pm"}, {"18:00", "6 pm"}, {"19:00", "7 pm"}, {"20:00", "8 pm"}, {"22:00", "10 pm"}, {"23:00", "11 pm"}, {"off", "Off for everyone"}}
var reminderLevels = []Option{{"", "Below 60% (default)"}, {"30", "Below 30%"}, {"40", "Below 40%"}, {"50", "Below 50%"}, {"70", "Below 70%"}, {"80", "Below 80%"}, {"101", "Any level"}}

func init() {
	Defs = append(Defs,
		Def{Key: PlugReminderTimeKey, Label: "Plug-in reminder time", Options: reminderTimes,
			Help: "If the car is parked, not plugged in and low enough, devices that turned on the Plug in tonight alert get one reminder at this time (in your time zone). At most one a day."},
		Def{Key: PlugReminderBelowKey, Label: "Plug-in reminder battery level", Options: reminderLevels,
			Help: "The reminder is sent only when the battery is below this level."})
	validators[PlugReminderTimeKey] = func(v string) error {
		for _, o := range reminderTimes {
			if o.Value == v {
				return nil
			}
		}
		return fmt.Errorf("choose one of the times")
	}
	validators[PlugReminderBelowKey] = func(v string) error {
		for _, o := range reminderLevels {
			if o.Value == v {
				return nil
			}
		}
		return fmt.Errorf("choose one of the levels")
	}
}

// PlugReminderTime is the time of day as "21:00", or "" when an admin turned the reminder off.
func (s *Settings) PlugReminderTime() string {
	switch v := s.Value(PlugReminderTimeKey); v {
	case "":
		return "21:00"
	case "off":
		return ""
	default:
		return v
	}
}

// PlugReminderBelow is the battery percent under which the reminder applies.
func (s *Settings) PlugReminderBelow() float64 {
	if n, err := strconv.Atoi(s.Value(PlugReminderBelowKey)); err == nil {
		return float64(n)
	}
	return 60
}
