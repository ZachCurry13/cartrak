// Package rebuild redoes the trips and charging sessions of a stretch of time from the readings that were stored,
// with the current detection rules. It is the repair for a day when trips were recorded wrongly. Nothing outside
// the stretch is touched, notes, hand-chosen categories and typed-in charger kWh are carried over, and the swap
// happens in one step or not at all.
package rebuild

import (
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/store"
	"github.com/ZachCurry13/cartrak/internal/trips"
)

const (
	maxSpan   = 7 * 24 * time.Hour
	settleFor = 30 * time.Minute // the end of the stretch must be at least this long ago
)

// Plan is what a rebuild would replace and what it would put there.
type Plan struct {
	From       int64           `json:"from"` // the stretch, widened to whole trips and charges
	To         int64           `json:"to"`
	Readings   int             `json:"readings"`
	OldTrips   []*store.Trip   `json:"oldTrips"`
	OldCharges []*store.Charge `json:"oldCharges"`
	NewTrips   []*store.Trip   `json:"newTrips"`
	NewCharges []*store.Charge `json:"newCharges"`
	Blockers   []string        `json:"blockers"`
}

func end(t *store.Trip) int64 {
	if t.EndTS != nil {
		return *t.EndTS
	}
	return t.StartTS
}

func cEnd(c *store.Charge) int64 {
	if c.EndTS != nil {
		return *c.EndTS
	}
	return c.StartTS
}

// Compute works out the plan for [from, to] without changing anything.
func Compute(s *store.Store, d *trips.Detector, from, to int64, now time.Time) (*Plan, error) {
	p := &Plan{From: from, To: to}
	if to <= from {
		p.Blockers = append(p.Blockers, "The end must be after the start.")
		return p, nil
	}
	if time.Duration(to-from)*time.Second > maxSpan {
		p.Blockers = append(p.Blockers, "Choose a stretch of seven days or less.")
		return p, nil
	}
	if time.Unix(to, 0).After(now.Add(-settleFor)) {
		p.Blockers = append(p.Blockers, "The stretch must end at least 30 minutes ago, so a trip in progress is never rebuilt.")
		return p, nil
	}
	// Widen to whole trips and charging sessions that overlap the stretch.
	for i := 0; i < 5; i++ {
		ts, err := s.TripsTouching(p.From, p.To)
		if err != nil {
			return nil, err
		}
		cs, err := s.ChargesTouching(p.From, p.To)
		if err != nil {
			return nil, err
		}
		p.OldTrips, p.OldCharges = ts, cs
		f, t := p.From, p.To
		for _, x := range ts {
			f, t = min(f, x.StartTS), max(t, end(x))
		}
		for _, x := range cs {
			f, t = min(f, x.StartTS), max(t, cEnd(x))
		}
		if f == p.From && t == p.To {
			break
		}
		p.From, p.To = f, t
	}
	for _, t := range p.OldTrips {
		if t.Open {
			p.Blockers = append(p.Blockers, fmt.Sprintf("Trip %d is still open. Close it first (open the trip, Fix this trip).", t.ID))
		}
		if t.JourneyID != nil {
			p.Blockers = append(p.Blockers, fmt.Sprintf("Trip %d is part of a journey. Delete that journey first (it only groups trips), then rebuild.", t.ID))
		}
	}
	for _, c := range p.OldCharges {
		if c.Open {
			p.Blockers = append(p.Blockers, "A charging session is still in progress in that stretch.")
		}
	}
	if time.Duration(p.To-p.From)*time.Second > maxSpan {
		p.Blockers = append(p.Blockers, "Once widened to whole trips the stretch is longer than seven days.")
	}
	if len(p.Blockers) > 0 {
		return p, nil
	}
	rows, err := s.ReplayRows(p.From, p.To)
	if err != nil {
		return nil, err
	}
	p.Readings = len(rows)
	if len(rows) == 0 {
		p.Blockers = append(p.Blockers, "No readings are stored for that time.")
		return p, nil
	}
	nt, nc, err := replay(rows, d)
	if err != nil {
		return nil, err
	}
	p.NewTrips, p.NewCharges = nt, nc
	return p, nil
}

// replay runs the detector over the readings in a scratch database and returns what it recorded.
func replay(rows []store.ReplayRow, d *trips.Detector) ([]*store.Trip, []*store.Charge, error) {
	dir, err := os.MkdirTemp("", "cartrak-rebuild-")
	if err != nil {
		return nil, nil, err
	}
	defer os.RemoveAll(dir)
	tmp, err := store.Open(dir)
	if err != nil {
		return nil, nil, err
	}
	defer tmp.Close()
	det := &trips.Detector{S: tmp, TZ: d.TZ, TZFn: d.TZFn}
	for _, r := range rows {
		st := &ford.State{TS: time.Unix(r.TS, 0), OdometerKm: r.OdometerKm, SoCPct: r.SoC, Lat: r.Lat, Lon: r.Lon, SpeedKph: r.SpeedKph,
			Ignition: r.Ignition, ChargeState: r.ChargeState, ChargeKw: r.ChargeKw, EnergyKwh: r.EnergyKwh, Plug: r.Plug}
		if err := det.Process(st); err != nil {
			return nil, nil, err
		}
	}
	if t, _ := tmp.OpenTrip(); t != nil { // the stretch ends at a boundary, so anything still open is finished
		t.Open = false
		if err := tmp.UpdateTrip(t); err != nil {
			return nil, nil, err
		}
	}
	if c, _ := tmp.OpenCharge(); c != nil {
		c.Open = false
		if err := tmp.UpdateCharge(c); err != nil {
			return nil, nil, err
		}
	}
	nt, err := tmp.ListTrips(store.TripFilter{Limit: 1000})
	if err != nil {
		return nil, nil, err
	}
	nc, err := tmp.ListCharges(1000)
	if err != nil {
		return nil, nil, err
	}
	sort.Slice(nt, func(i, j int) bool { return nt[i].StartTS < nt[j].StartTS })
	sort.Slice(nc, func(i, j int) bool { return nc[i].StartTS < nc[j].StartTS })
	return nt, nc, nil
}

func overlap(a0, a1, b0, b1 int64) int64 { return max(0, min(a1, b1)-max(a0, b0)) }

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
