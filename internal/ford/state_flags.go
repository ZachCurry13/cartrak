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

// PlugConnected reports whether the reading says a charging cable is connected.
func (s *State) PlugConnected() bool { return plugConnected(s.Plug) }

// Charging reports whether a reading shows the car taking charge.
func (s *State) Charging() bool {
	return ChargeStateCharging(s.ChargeState) || (s.ChargeKw != nil && *s.ChargeKw > 0.2)
}

// ChargeStateCharging reads the car's charge status word. Ford says IN_PROGRESS while charging (fast chargers report
// no power at all, so this is the only sign), and older feeds say CHARGING. NOT_READY, PAUSED, FAULT, COMPLETE and
// the like are not charging.
func ChargeStateCharging(state string) bool {
	u := strings.ToUpper(state)
	if strings.Contains(u, "NOT") || strings.Contains(u, "PAUSE") || strings.Contains(u, "COMPLETE") || strings.Contains(u, "FAULT") || strings.Contains(u, "STOP") {
		return false
	}
	return strings.Contains(u, "IN_PROGRESS") || strings.Contains(u, "CHARGING")
}

// plugConnected is true when the plug state says a cable is connected (and not "disconnected").
func plugConnected(plug string) bool {
	p := strings.ToLower(plug)
	return strings.Contains(p, "connect") && !strings.Contains(p, "dis") && !strings.Contains(p, "not")
}
