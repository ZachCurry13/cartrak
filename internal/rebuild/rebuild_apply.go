package rebuild

import (
	"fmt"
	"sort"

	"github.com/ZachCurry13/cartrak/internal/store"
	"github.com/ZachCurry13/cartrak/internal/trips"
)

// Apply swaps the old trips and charges of the plan for the new ones. It refuses a plan with blockers.
func Apply(s *store.Store, d *trips.Detector, p *Plan) error {
	if len(p.Blockers) > 0 {
		return fmt.Errorf("%s", p.Blockers[0])
	}
	// The trips of the day before the stretch, so an outing that began earlier is known; each new trip is added as it is classified.
	earlier := tripsBefore(s, p.From)
	for _, n := range p.NewTrips {
		var best *store.Trip
		var bestOv int64
		for _, o := range p.OldTrips {
			if ov := overlap(n.StartTS, end(n), o.StartTS, end(o)); ov > bestOv {
				best, bestOv = o, ov
			}
		}
		d.ClassifyTripAfter(n, earlier)
		if best != nil {
			n.Notes = best.Notes
			// a distance the owner corrected carries over when the new trip is the same drive, not a merge or a split of it
			if best.DistanceSrc != "" && best.EndTS != nil && bestOv*10 >= (*best.EndTS-best.StartTS)*8 && bestOv*10 >= (end(n)-n.StartTS)*8 {
				n.DistanceKm, n.DistanceSrc = best.DistanceKm, best.DistanceSrc
			}
			if best.CategorySource == "manual" {
				n.Category, n.CategorySource, n.RuleID = best.Category, "manual", nil
			}
		}
		earlier = append(earlier, n)
	}
	for _, n := range p.NewCharges {
		var best *store.Charge
		var bestOv int64
		for _, o := range p.OldCharges {
			if ov := overlap(n.StartTS, cEnd(n), o.StartTS, cEnd(o)); ov > bestOv {
				best, bestOv = o, ov
			}
		}
		if best != nil {
			n.Kind, n.ActualCost = best.Kind, best.ActualCost
			if best.ChargerKwh != nil { // a figure typed in beats the car's own count that the replay may have found
				n.ChargerKwh = best.ChargerKwh
			}
			if n.Station == "" {
				n.Station = best.Station
			}
		}
	}
	var ot, oc []int64
	for _, t := range p.OldTrips {
		ot = append(ot, t.ID)
	}
	for _, c := range p.OldCharges {
		oc = append(oc, c.ID)
	}
	return s.ReplaceRange(ot, oc, p.NewTrips, p.NewCharges)
}

// Fill turns missing lists into empty ones so the plan always serialises as lists, never null.
func (p *Plan) Fill() {
	if p.OldTrips == nil {
		p.OldTrips = []*store.Trip{}
	}
	if p.OldCharges == nil {
		p.OldCharges = []*store.Charge{}
	}
	if p.NewTrips == nil {
		p.NewTrips = []*store.Trip{}
	}
	if p.NewCharges == nil {
		p.NewCharges = []*store.Charge{}
	}
	if p.Blockers == nil {
		p.Blockers = []string{}
	}
}

// tripsBefore are the stored trips in the day and a half before ts, oldest first.
func tripsBefore(s *store.Store, ts int64) []*store.Trip {
	ps, err := s.ListTrips(store.TripFilter{From: ts - 36*3600, To: ts, All: true})
	if err != nil {
		return []*store.Trip{}
	}
	sort.Slice(ps, func(i, j int) bool { return ps[i].StartTS < ps[j].StartTS })
	if ps == nil {
		ps = []*store.Trip{}
	}
	return ps
}
