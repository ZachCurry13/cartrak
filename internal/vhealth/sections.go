package vhealth

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/ZachCurry13/cartrak/internal/ford"
)

var camel = regexp.MustCompile(`([a-z0-9])([A-Z])`)

// words turns FRONT_LEFT or dieselParticulateFilter into "Front left" / "Diesel particulate filter".
func words(s string) string {
	t := strings.ToLower(strings.ReplaceAll(camel.ReplaceAllString(s, "$1 $2"), "_", " "))
	if t == "" {
		return ""
	}
	return strings.ToUpper(t[:1]) + t[1:]
}

// wordy makes a text row's value readable (NORMAL_OPERATION becomes "Normal operation").
func (r *Row) wordy() *Row {
	if r != nil {
		r.Text = words(r.Text)
	}
	return r
}

func get(it map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := it[k]; ok {
			return fmt.Sprint(v)
		}
	}
	return ""
}

func firstOf(it map[string]any, keys ...string) any {
	for _, k := range keys {
		if v, ok := it[k]; ok {
			return v
		}
	}
	return nil
}

// attention lists trouble codes, Ford alerts and warning lights. Missing means none reported.
func attention(f map[string]any) Section {
	s := Section{ID: "warnings", Title: "Warnings and codes",
		Intro: "What the car and Ford report right now. Ford sends a code or warning light only while there is one."}
	codes := ford.ListItems(f, "diagnosticTroubleCode")
	if len(codes) == 0 {
		if v := ford.Text(f, "diagnosticTroubleCode"); v != "" {
			codes = append(codes, map[string]any{"value": v})
		}
	}
	for _, c := range codes {
		note := ""
		if sys := get(c, "system", "dtcsystem"); sys != "" {
			note = words(sys) + " system"
		}
		s.Rows = append(s.Rows, Row{Label: "Trouble code", Kind: "text", Text: get(c, "value", "dtcvalue", "code"), State: "bad", Note: note})
	}
	if len(codes) == 0 {
		s.Rows = append(s.Rows, Row{Label: "Trouble codes", Kind: "text", Text: "None reported", State: "ok"})
	}
	alerts := 0
	for _, a := range ford.ListItems(f, "VehicleAlertList") {
		desc := get(a, "statusdesc", "description")
		if desc == "" || strings.EqualFold(desc, "No Data") || get(a, "statuscode") == "4040" {
			continue // Ford fills the list with a "No Data" line when there are no alerts
		}
		alerts++
		s.Rows = append(s.Rows, Row{Label: "Ford alert", Kind: "text", Text: desc, State: "warn", At: get(a, "eventtimestamp")})
	}
	if alerts == 0 {
		s.Rows = append(s.Rows, Row{Label: "Vehicle alerts", Kind: "text", Text: "None", State: "ok"})
	}
	lights := ford.Indicators(f)
	for _, l := range lights {
		s.Rows = append(s.Rows, Row{Label: "Warning light", Kind: "text", Text: words(l), State: "bad"})
	}
	if len(lights) == 0 {
		s.Rows = append(s.Rows, Row{Label: "Warning lights", Kind: "text", Text: "None on", State: "ok"})
	}
	return s
}

// traction is the large drive battery of an electric or hybrid car.
func traction(f map[string]any, t Trends) Section {
	shown := num(f, "Charge shown on the dash", "pct", "xevBatteryStateOfCharge")
	actual := num(f, "Actual charge", "pct", "xevBatteryActualStateOfCharge")
	if actual != nil && shown != nil && *shown.Num != *actual.Num {
		actual.Note = "The true charge of the battery cells. It includes a small buffer at the top and bottom that the dash hides to protect the battery, so the dash percentage is scaled to the part you can use."
	}
	capKwh := num(f, "Estimated capacity", "kwh", "xevBatteryCapacity")
	if capKwh != nil {
		capKwh.Note = "The car's own estimate of what the battery can hold now. It moves a few percent with the temperature and the charge level, so the chart shows the middle reading of each day. Watch it over months: a steady fall is normal ageing."
	}
	status := text(f, "Battery status", "xevBatteryPerformanceStatus")
	if status != nil {
		status.State = "ok"
		if !strings.EqualFold(status.Text, "NORMAL") {
			status.State = "warn"
		}
		status.wordy()
	}
	return Section{ID: "traction", Title: "Main battery", Rows: rows(shown, actual,
		num(f, "Energy left", "kwh", "xevBatteryEnergyRemaining"), capKwh, status,
		num(f, "Voltage", "volts", "xevBatteryVoltage"), num(f, "Temperature", "tempC", "xevBatteryTemperature"),
		num(f, "Current", "amps", "xevBatteryIoCurrent"), num(f, "Range now", "km", "xevBatteryRange"),
		num(f, "Range when full", "km", "xevBatteryMaximumRange"), num(f, "Drive motor voltage", "volts", "xevTractionMotorVoltage")),
		Series: series(t, "capacityKwh", "Estimated capacity, middle reading of each day", "kwh")}
}

// engine is for cars that burn fuel: oil life, fuel and engine temperatures.
func engine(f map[string]any, pt ford.Powertrain, t Trends) Section {
	oil := num(f, "Oil life left", "pct", "oilLifeRemaining")
	if oil != nil {
		switch {
		case *oil.Num <= 5:
			oil.with("bad", "Change the oil now.")
		case *oil.Num <= 15:
			oil.with("warn", "Plan an oil change soon.")
		default:
			oil.with("ok", "")
		}
	}
	fuel := num(f, "Fuel level", "pct", "fuelLevel")
	if fuel != nil && *fuel.Num <= 10 {
		fuel.with("warn", "Low fuel")
	}
	var mode *Row
	if pt != ford.Gas {
		mode = text(f, "Hybrid mode", "hybridVehicleModeStatus").wordy()
	}
	return Section{ID: "engine", Title: "Engine and oil", Rows: rows(oil, fuel, num(f, "Fuel range", "km", "fuelRange"), mode,
		num(f, "Coolant temperature", "tempC", "engineCoolantTemp"), num(f, "Oil temperature", "tempC", "engineOilTemp"),
		num(f, "Particulate filter soot", "pct", "dieselParticulateFilterSootLevel")),
		Series: series(t, "oilLifePct", "Oil life left, daily average", "pct")}
}

// tires lists each tire's pressure against its placard value and the car's own status.
func tires(f map[string]any) Section {
	status := map[string]string{}
	for _, it := range ford.ListItems(f, "tirePressureStatus") {
		status[get(it, "vehiclewheel")] = strings.ToUpper(get(it, "value"))
	}
	s := Section{ID: "tires", Title: "Tires"}
	for _, it := range ford.ListItems(f, "tirePressure") {
		wheel := get(it, "vehiclewheel")
		v := ford.Number(map[string]any{"v": it["value"]}, "v")
		if wheel == "" || v == nil {
			continue
		}
		r := Row{Label: words(wheel), Kind: "kpa", Num: v, State: "ok", At: get(it, "updatetime")}
		if st := status[wheel]; st != "" && st != "NORMAL" {
			r.State, r.Note = "warn", "The car reports "+strings.ToLower(words(st))+"."
		} else if p := ford.Number(map[string]any{"p": firstOf(it, "wheelplacardfront", "wheelplacardrear")}, "p"); p != nil && *v < *p*0.85 {
			r.State, r.Note = "warn", "Well below the recommended pressure on the door sticker."
		}
		s.Rows = append(s.Rows, r)
	}
	if sys := text(f, "Tire monitor", "tirePressureSystemStatus").wordy(); sys != nil && len(s.Rows) > 0 {
		s.Rows = append(s.Rows, *sys)
	}
	return s
}

// system is the rest: updates, alarm, parking brake and remote start state.
func system(f map[string]any) Section {
	return Section{ID: "system", Title: "Car systems", Rows: rows(
		text(f, "Automatic software updates", "automaticSoftwareUpdateOptInSetting").wordy(),
		text(f, "Alarm", "alarmStatus").wordy(), text(f, "Parking brake", "parkingBrakeStatus").wordy(),
		text(f, "Car mode", "vehicleLifeCycleMode").wordy(), num(f, "Remote start time left", "seconds", "remoteStartCountdownTimer"))}
}
