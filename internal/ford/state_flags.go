package ford

import "strings"

// Driving reports whether a reading looks like the car is in use.
func (s *State) Driving() bool {
	switch strings.ToUpper(s.Ignition) {
	case "ON", "RUN", "START", "STARTING", "RUNNING":
		return true
	}
	return s.SpeedKph != nil && *s.SpeedKph > 3
}

// Charging reports whether a reading shows the car taking charge.
func (s *State) Charging() bool {
	return strings.Contains(strings.ToLower(s.ChargeState), "charging") || (s.ChargeKw != nil && *s.ChargeKw > 0.2)
}
