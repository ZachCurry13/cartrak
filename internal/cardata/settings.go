package cardata

import (
	"fmt"
	"strings"

	"github.com/ZachCurry13/cartrak/internal/ford"
)

// settings is the Settings and schedules tab. CarTrak can read these but not change them: Ford's
// connection is read-only, so changes are made in the FordPass app or the car.
func (s src) settings() Tab {
	updates := group("Software updates", "The car can install updates by itself overnight when this is on.",
		s.text("Automatic updates", "Whether the car installs updates by itself.", "automaticSoftwareUpdateOptInSetting"),
		s.updateWindow())
	departures := group("Departure times", "Times you told the car you plan to leave, so it can finish charging and warm or cool the cabin first. Shown here only; change them in the FordPass app.",
		s.text("Departure times feature", "Whether the departure times feature is switched on.", "departureScheduleFeatureStatus"),
		s.departureSummary())
	if d := s.departureRows(); len(d) > 0 && departures != nil {
		departures.Rows = append(departures.Rows, d...)
	}
	other := group("Other settings", "",
		s.text("Units on the car's screen", "Whether the car's own screen uses miles or kilometres.", "displaySystemOfMeasure"),
		s.num("Remote start run time", "duration", "How long a remote start keeps the car running before it shuts off if you do not get in.", "remoteStartRunDurationSetting"),
		s.text("Car mode", "Normal means the car is in everyday use. Other modes are for the factory or transport.", "vehicleLifeCycleMode"))
	return tab("settings", "Settings and schedules", "What the car is set to do. Read only.", updates, departures, other)
}

// updateWindow says when updates install: "Every day at 1:00 AM" or one line per day.
func (s src) updateWindow() *Row {
	days := ford.ListItems(s.f, "dayOfWeekAndTime")
	if len(days) == 0 {
		return nil
	}
	same := true
	for _, d := range days {
		if fmt.Sprint(d["timeofday"]) != fmt.Sprint(days[0]["timeofday"]) {
			same = false
		}
	}
	if same && len(days) == 7 {
		return say("Update time", "Every day at "+clock12(fmt.Sprint(days[0]["timeofday"])), "When the car is allowed to install updates.")
	}
	var parts []string
	for _, d := range days {
		parts = append(parts, dayName(fmt.Sprint(d["dayofweek"]))+" "+clock12(fmt.Sprint(d["timeofday"])))
	}
	return say("Update times", strings.Join(parts, ", "), "When the car is allowed to install updates.")
}

type slot struct{ day, time, status string }

func (s src) slots() []slot {
	var out []slot
	for _, it := range ford.ListItems(s.f, "departureSchedules") {
		out = append(out, slot{fmt.Sprint(it["schedule.weeklyschedule.dayofweek"]), fmt.Sprint(it["schedule.weeklyschedule.timeofday"]), strings.ToUpper(fmt.Sprint(it["schedulestatus"]))})
	}
	return out
}

func (s src) departureSummary() *Row {
	all := s.slots()
	if len(all) == 0 {
		return nil
	}
	on := 0
	for _, x := range all {
		if x.status == "ON" {
			on++
		}
	}
	text := plural(on, "is", "are") + " turned on of " + plural(len(all), "slot", "slots")
	if on == 0 {
		text = "None turned on (" + plural(len(all), "slot is", "slots are") + " set up but off)"
	}
	return say("Departure times", text, "A slot is one saved departure time. Only slots that are on do anything.")
}

// departureRows lists the departure times that are turned on.
func (s src) departureRows() []Row {
	var rows []Row
	for _, x := range s.slots() {
		if x.status == "ON" {
			rows = append(rows, Row{Label: dayName(x.day), Kind: "text", Text: clock12(x.time) + " (on)", Raw: "departureSchedules"})
		}
	}
	return rows
}
