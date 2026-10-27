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
	// Each new trip is classified knowing the trips before it: the ones that stay (the stretch is widened to whole trips, and a trip
	// that already matches is left alone, so it is neither old nor new) and the new ones classified so far.
	fixed := keptTrips(s, p)
	var news []*store.Trip
	for _, n := range p.NewTrips {
		var best *store.Trip
		var bestOv int64
		for _, o := range p.OldTrips {
			if ov := overlap(n.StartTS, end(n), o.StartTS, end(o)); ov > bestOv {
				best, bestOv = o, ov
			}
		}
		d.ClassifyTripAfter(n, before(fixed, news, n.StartTS))
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
		news = append(news, n)
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
	if err := s.ReplaceRange(ot, oc, p.NewTrips, p.NewCharges); err != nil {
		return err
	}
	return s.FillTripStarts() // the replay had no positions of its own to start the new trips from
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

// keptTrips are the stored trips in the day and a half up to the end of the stretch that the plan does not replace, oldest first.
func keptTrips(s *store.Store, p *Plan) []*store.Trip {
	ps, err := s.ListTrips(store.TripFilter{From: p.From - 36*3600, To: p.To + 1, All: true})
	if err != nil {
		return []*store.Trip{}
	}
	gone := map[int64]bool{}
	for _, o := range p.OldTrips {
		gone[o.ID] = true
	}
	out := []*store.Trip{}
	for _, t := range ps {
		if !gone[t.ID] {
			out = append(out, t)
		}
	}
	return out
}

// before are the trips that start before ts, oldest first: the kept ones and the new ones so far.
func before(kept, news []*store.Trip, ts int64) []*store.Trip {
	out := []*store.Trip{}
	for _, list := range [][]*store.Trip{kept, news} {
		for _, t := range list {
			if t.StartTS < ts {
				out = append(out, t)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartTS < out[j].StartTS })
	return out
}
