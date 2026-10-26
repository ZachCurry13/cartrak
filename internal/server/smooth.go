package server

import (
	"encoding/json"
	"math"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/smooth"
	"github.com/ZachCurry13/cartrak/internal/store"
	"github.com/ZachCurry13/cartrak/internal/tripwhy"
)

// driveTorque is the torque readings of a trip while the car was moving.
func (s *Server) driveTorque(t *store.Trip) []smooth.Point {
	if t.EndTS == nil {
		return nil
	}
	rows, err := s.S.SamplesBetween(t.StartTS, *t.EndTS)
	if err != nil {
		return nil
	}
	var out []smooth.Point
	for _, x := range rows {
		if x.SpeedKph == nil || *x.SpeedKph < 3 {
			continue
		}
		f := map[string]any{}
		if json.Unmarshal(x.Extra, &f) != nil {
			continue
		}
		// The accelerator pedal says how hard the driver pushed far more finely than the torque field does (hundreds of values against a few
		// dozen), so it is used when the car sends it. Braking counts the friction brakes and also regeneration, which shows as negative torque.
		tq := ford.Number(f, "torqueAtTransmission")
		pedal := ford.Number(f, "acceleratorPedalPosition")
		if tq == nil && pedal == nil {
			continue
		}
		p := smooth.Point{}
		switch {
		case pedal != nil:
			p.Accel = max(0, *pedal)
		case *tq > 0:
			p.Accel = *tq
		}
		if tq != nil && *tq < 0 {
			p.Brake = -*tq
		}
		if b := ford.Number(f, "brakeTorque"); b != nil {
			p.Brake = max(p.Brake, math.Abs(*b))
		}
		out = append(out, p)
	}
	return out
}

// smoothBaseline is what a firm reading is for this driver, from the last dozen trips, kept for an hour.
func (s *Server) smoothBaseline() smooth.Baseline {
	s.cc.smoothMu.Lock()
	defer s.cc.smoothMu.Unlock()
	if time.Since(s.cc.smoothAt) < time.Hour {
		return s.cc.smoothBase
	}
	var accel, brake []float64
	n := 0
	list, _ := s.S.ListTrips(store.TripFilter{Limit: 12})
	for _, t := range list {
		if t.Open || t.EndTS == nil {
			continue
		}
		pts := s.driveTorque(t)
		if len(pts) == 0 {
			continue
		}
		n++
		for _, p := range pts {
			accel = append(accel, p.Accel)
			brake = append(brake, p.Brake)
		}
	}
	s.cc.smoothBase = smooth.Baseline{Accel: smooth.Percentile90(accel), Brake: smooth.Percentile90(brake), Trips: n}
	s.cc.smoothAt = time.Now()
	return s.cc.smoothBase
}

// withSmoothness adds how firmly the trip was driven, and a short reason when it matters: called bad or good only when the
// trip's energy use backs it up, the way the other reasons are.
func (s *Server) withSmoothness(rep *tripwhy.Report, t *store.Trip) {
	b := s.smoothBaseline()
	if b.Trips < 3 {
		return
	}
	r, ok := smooth.Judge(s.driveTorque(t), b)
	if !ok {
		return
	}
	rep.Smooth = &r
	switch {
	case r.Verdict == "firm":
		lvl := "info"
		if rep.Ready && rep.VsUsualPct > 3 {
			lvl = "bad"
		}
		rep.Tags = append(rep.Tags, tripwhy.Tag{Key: "firm", Label: "Firm driving", Level: lvl, Bad: lvl == "bad"})
	case r.Verdict == "smooth":
		lvl := "info"
		if rep.Ready && rep.VsUsualPct < -3 {
			lvl = "good"
		}
		rep.Tags = append(rep.Tags, tripwhy.Tag{Key: "smooth", Label: "Smooth driving", Level: lvl})
	}
}
