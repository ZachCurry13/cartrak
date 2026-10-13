package cardata

import (
	"fmt"
	"strings"

	"github.com/ZachCurry13/cartrak/internal/ford"
)

// car is the Car tab: what this car is, and what Ford shares about it. It never shows the VIN.
func (s src) car() Tab {
	var model *Row
	year, name := ford.Text(s.f, "garage.modelYear"), ford.Text(s.f, "garage.modelName")
	if name != "" {
		model = say("Model", strings.TrimSpace(year+" "+name), "The model Ford has on record for this car.")
	}
	about := group("About this car", "",
		model,
		s.make(),
		s.plain("Trim code", "Ford's short code for this version of the car.", "garage.modelCode"),
		s.text("Colour", "The paint colour on record.", "garage.color"),
		s.text("Vehicle type", "The kind of vehicle on record.", "garage.vehicleType"),
		s.text("Engine type", "How the car is powered. BEV means battery electric.", "garage.engineType"),
		say("Treated by CarTrak as", s.pt.Label(), "CarTrak uses this to decide which batteries, fuel and charging details to show. It comes from Ford's record, or your choice in Settings."),
		s.yes("Built-in modem on", "Whether the car's modem is switched on. It is how the car talks to Ford.", "garage.tcuEnabled"),
		s.yes("Access approved", "Whether you have approved connected-vehicle access for this car, which is what lets CarTrak read it.", "garage.vehicleAuthorizationIndicator"))
	data := group("What Ford shares", "",
		say("Values in this reading", fmt.Sprintf("%d separate values", len(s.f)), "How many individual numbers and words Ford sent in this one reading."),
		s.plain("Last update from Ford", "When Ford last refreshed this car's data.", "telemetry.updateTime"),
		say("Connection", "FordConnect Query (read only)", "CarTrak can read your car's data but cannot send it commands such as locking or starting."))
	return tab("car", "Car", "About this car and the data.", about, data)
}

func (s src) make() *Row {
	m := ford.Text(s.f, "garage.make")
	switch strings.ToUpper(m) {
	case "":
		return nil
	case "F":
		m = "Ford"
	case "L":
		m = "Lincoln"
	}
	return say("Make", m, "The brand.")
}

// yes shows a 1/0 flag as Yes/No.
func (s src) yes(label, help, name string) *Row {
	v := ford.Number(s.f, name)
	if v == nil {
		return nil
	}
	text := "No"
	if *v != 0 {
		text = "Yes"
	}
	return say(label, text, help)
}
