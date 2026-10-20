package rebuild

import (
	"math"

	"github.com/ZachCurry13/cartrak/internal/store"
)

// A rebuild only touches what would come out different. A trip or charge that the replay records exactly as it is stored is left
// alone, so its id, its public links, its notes and its labels are not disturbed just because it was in the stretch.

func sameF(a, b *float64) bool {
	if a == nil || b == nil {
		return a == b
	}
	return math.Abs(*a-*b) < 1e-6
}

func sameI(a, b *int64) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// sameTrip compares what detection decides. The category and notes are the owner's or the rules', so they are not compared, and a
// distance the owner corrected is not compared either.
func sameTrip(o, n *store.Trip) bool {
	if o.StartTS != n.StartTS || !sameI(o.EndTS, n.EndTS) || o.Open != n.Open {
		return false
	}
	if o.DistanceSrc == "" && math.Abs(o.DistanceKm-n.DistanceKm) > 1e-6 {
		return false
	}
	return sameF(o.StartLat, n.StartLat) && sameF(o.StartLon, n.StartLon) && sameF(o.EndLat, n.EndLat) && sameF(o.EndLon, n.EndLon) &&
		sameF(o.StartOdo, n.StartOdo) && sameF(o.EndOdo, n.EndOdo) && sameF(o.StartSoC, n.StartSoC) && sameF(o.EndSoC, n.EndSoC) &&
		sameF(o.StartKwh, n.StartKwh) && sameF(o.EndKwh, n.EndKwh) && math.Abs(o.MaxSpeedKph-n.MaxSpeedKph) < 1e-6
}

// sameCharge compares what detection decides from the stored readings. What the owner typed (kind, cost, the charger's kWh) is not
// compared, and neither is the charger type, which the replay does not see (a rebuild keeps the stored one).
func sameCharge(o, n *store.Charge) bool {
	if o.StartTS != n.StartTS || !sameI(o.EndTS, n.EndTS) || o.Open != n.Open {
		return false
	}
	return sameF(o.Lat, n.Lat) && sameF(o.Lon, n.Lon) && sameF(o.StartSoC, n.StartSoC) && sameF(o.EndSoC, n.EndSoC) &&
		sameF(o.StartKwh, n.StartKwh) && sameF(o.EndKwh, n.EndKwh) && sameF(o.PowerKwh, n.PowerKwh) && math.Abs(o.PeakKw-n.PeakKw) < 1e-6
}

// dropUnchanged removes the pairs that came out the same from both sides of the plan and counts them.
func (p *Plan) dropUnchanged() {
	keepOldT, keepNewT := make([]*store.Trip, 0, len(p.OldTrips)), make([]*store.Trip, 0, len(p.NewTrips))
	same := map[*store.Trip]bool{}
	for _, n := range p.NewTrips {
		found := false
		for _, o := range p.OldTrips {
			if !same[o] && sameTrip(o, n) {
				same[o], found = true, true
				p.UnchangedTrips++
				break
			}
		}
		if !found {
			keepNewT = append(keepNewT, n)
		}
	}
	for _, o := range p.OldTrips {
		if !same[o] {
			keepOldT = append(keepOldT, o)
		}
	}
	p.OldTrips, p.NewTrips = keepOldT, keepNewT

	keepOldC, keepNewC := make([]*store.Charge, 0, len(p.OldCharges)), make([]*store.Charge, 0, len(p.NewCharges))
	sameC := map[*store.Charge]bool{}
	for _, n := range p.NewCharges {
		found := false
		for _, o := range p.OldCharges {
			if !sameC[o] && sameCharge(o, n) {
				sameC[o], found = true, true
				p.UnchangedCharges++
				break
			}
		}
		if !found {
			keepNewC = append(keepNewC, n)
		}
	}
	for _, o := range p.OldCharges {
		if !sameC[o] {
			keepOldC = append(keepOldC, o)
		}
	}
	p.OldCharges, p.NewCharges = keepOldC, keepNewC
}
