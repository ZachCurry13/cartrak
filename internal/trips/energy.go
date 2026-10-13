package trips

import "github.com/ZachCurry13/cartrak/internal/store"

// Energy is how much battery energy a trip used, and how that was worked out.
type Energy struct {
	Kwh       float64
	Estimated bool // from the battery percentage and capacity, not the car's own energy reading
}

const (
	minEnergyKwh  = 0.05 // below this the readings only differ by noise
	minSoCDropPct = 0.5
)

// EnergyUsed works out the energy a trip used. It prefers the difference in the car's own
// energy-left reading between the start and the end. For trips recorded without it, it falls back to
// the battery percentage drop times the battery's estimated capacity (capacityKwh, may be nil).
// It reports false when neither can be worked out, for example when the trip gained charge.
func EnergyUsed(t *store.Trip, capacityKwh *float64) (Energy, bool) {
	if t.StartKwh != nil && t.EndKwh != nil {
		if used := *t.StartKwh - *t.EndKwh; used >= minEnergyKwh {
			return Energy{Kwh: used}, true
		}
		return Energy{}, false // regeneration or a stale reading: do not invent a figure
	}
	if t.StartSoC != nil && t.EndSoC != nil && capacityKwh != nil && *capacityKwh > 0 {
		if drop := *t.StartSoC - *t.EndSoC; drop >= minSoCDropPct {
			return Energy{Kwh: drop / 100 * *capacityKwh, Estimated: true}, true
		}
	}
	return Energy{}, false
}
