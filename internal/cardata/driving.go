package cardata

import (
	"strings"

	"github.com/ZachCurry13/cartrak/internal/ford"
)

// driving is the Driving tab: controls, motion sensors, direction and what the car recovered on this trip.
func (s src) driving() Tab {
	var speed *Row
	if ms := ford.Number(s.f, "speed"); ms != nil {
		kph := *ms * 3.6 // Ford sends meters per second
		speed = &Row{Label: "Speed", Kind: "kph", Num: &kph, Help: "How fast the car is going.", Raw: "speed", At: ford.Updated(s.f, "speed")}
	}
	controls := group("Controls", "",
		s.text("Gear", "Which gear the selector is in: park, reverse, neutral or drive.", "gearLeverPosition"),
		s.text("Ignition", "Off means the car is switched off. On means it is switched on, even if it is not moving.", "ignitionStatus"),
		speed,
		s.num("Accelerator pedal", "pct", "How far the accelerator pedal is pressed. Zero when your foot is off it.", "acceleratorPedalPosition"),
		s.text("Brake pedal", "Whether the brake pedal is being pressed.", "brakePedalStatus"),
		s.num("Braking force", "nm", "How hard the brakes are working, in newton-meters (Nm).", "brakeTorque"),
		s.num("Power at the wheels", "nm", "The pushing force the motor is delivering to the wheels, in newton-meters (Nm).", "torqueAtTransmission"),
		s.text("Wheel drive", "Whether power is reaching the wheels right now.", "wheelTorqueStatus"),
		s.parkingBrake())
	motion := group("Motion sensors", "Built-in sensors that feel how the car moves. They read close to zero while parked.",
		s.num("Forward and back", "accel", "Speeding up (positive) or slowing down (negative), in meters per second squared.", "acceleration.value.x"),
		s.num("Side to side", "accel", "Sideways push, for example in a turn, in meters per second squared.", "acceleration.value.y"),
		s.num("Up and down", "accel", "Up and down movement, for example over bumps, in meters per second squared.", "acceleration.value.z"),
		s.num("Turning rate", "degps", "How fast the car is turning, in degrees per second. Zero when going straight or parked.", "yawRate"))
	direction := group("Direction", "",
		s.num("Heading", "deg", "Which way the car is pointing. 0 is north, 90 is east, 180 is south and 270 is west.", "heading.value.heading"),
		s.text("Compass", "The same direction in words.", "compassDirection"))
	trip := group("Recovered by braking", "Slowing down with the motor turns it into a generator that puts energy back into the battery (regenerative braking). These are the car's own figures for the current trip.",
		s.num("Charge recovered", "pct", "How much battery charge was won back by regenerative braking on this trip, as a percent.", "tripXevBatteryChargeRegenerated"),
		s.num("Range recovered", "km", "How much range was won back by regenerative braking on this trip.", "tripXevBatteryRangeRegenerated"))
	return tab("driving", "Driving", "What the car is doing while it moves.", controls, motion, direction, trip)
}

// parkingBrake joins the state and the type: "Off (mechanical)".
func (s src) parkingBrake() *Row {
	r := s.text("Parking brake", "Whether the parking brake is on.", "parkingBrakeStatus")
	if r == nil {
		return nil
	}
	if t := ford.Text(s.f, "parkingBrakeStatus.parkingBrakeType"); t != "" {
		r.Text += " (" + lowerFirst(words(t)) + ")"
	}
	return r
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}
