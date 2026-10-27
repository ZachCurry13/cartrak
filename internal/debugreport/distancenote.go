package debugreport

import (
	"fmt"
	"github.com/ZachCurry13/cartrak/internal/store"
)

// distanceNote says where a trip's distance came from when it is not the odometer's own: corrected by hand, from the estimated road,
// or as the straight line, with what the odometer said. A trip nobody touched adds nothing.
func distanceNote(t *store.Trip, per float64, unit string) string {
	odo := ""
	if t.StartOdo != nil && t.EndOdo != nil && *t.EndOdo >= *t.StartOdo {
		if d := *t.EndOdo - *t.StartOdo; d-t.DistanceKm > 0.1 || t.DistanceKm-d > 0.1 {
			odo = fmt.Sprintf("the odometer said %.1f %s", d*per, unit)
		}
	}
	switch {
	case t.DistanceSrc != "" && odo != "":
		return fmt.Sprintf(" | distance corrected (%s): %s", t.DistanceSrc, odo)
	case t.DistanceSrc != "":
		return fmt.Sprintf(" | distance corrected (%s)", t.DistanceSrc)
	case odo != "":
		return " | " + odo + ", not corrected"
	}
	return ""
}
