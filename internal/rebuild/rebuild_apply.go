package rebuild

import (
	"fmt"

	"github.com/ZachCurry13/cartrak/internal/store"
	"github.com/ZachCurry13/cartrak/internal/trips"
)

// Apply swaps the old trips and charges of the plan for the new ones. It refuses a plan with blockers.
func Apply(s *store.Store, d *trips.Detector, p *Plan) error {
	if len(p.Blockers) > 0 {
		return fmt.Errorf("%s", p.Blockers[0])
	}
	for _, n := range p.NewTrips {
		var best *store.Trip
		var bestOv int64
		for _, o := range p.OldTrips {
			if ov := overlap(n.StartTS, end(n), o.StartTS, end(o)); ov > bestOv {
				best, bestOv = o, ov
			}
		}
		d.ClassifyTrip(n)
		if best != nil {
			n.Notes = best.Notes
			if best.CategorySource == "manual" {
				n.Category, n.CategorySource, n.RuleID = best.Category, "manual", nil
			}
		}
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
			n.Kind, n.ChargerKwh, n.ActualCost = best.Kind, best.ChargerKwh, best.ActualCost
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
