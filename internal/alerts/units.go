package alerts

import "fmt"

// distance is a distance in one unit: miles, or kilometers when the owner's units are metric. A notification shows one number,
// not both.
func (e *Engine) distance(km float64, places int) string {
	if e.Metric != nil && e.Metric() {
		return fmt.Sprintf("%.*f km", places, km)
	}
	return fmt.Sprintf("%.*f mi", places, km*0.621371)
}

// perEnergy is " 3.4 mi/kWh." (or km/kWh in metric) for the distance driven on that much energy.
func (e *Engine) perEnergy(km, kwh float64) string {
	if e.Metric != nil && e.Metric() {
		return fmt.Sprintf(" %.1f km/kWh.", km/kwh)
	}
	return fmt.Sprintf(" %.1f mi/kWh.", km*0.621371/kwh)
}
