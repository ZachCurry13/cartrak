// Package drain measures how much charge an electric car loses while it sits parked: not driving and
// not charging. A little is normal, because the car keeps its computers and its 12 V battery alive.
package drain

import (
	"github.com/ZachCurry13/cartrak/internal/store"
)

const (
	minStretchHours = 3    // shorter stops say little, and the battery level only changes in steps
	minTotalHours   = 12   // parked time needed before a daily rate means anything
	driveKm         = 0.3  // the odometer moving this much between readings means the car was driven
	chargeJumpPct   = 2.0  // a rise this big between readings was charging the car did not flag
	maxDropPct      = 30.0 // a bigger fall than this over a parked stretch is a data oddity, not drain
)

// Stretch is one stay parked, from the first reading of it to the last.
type Stretch struct {
	From    int64   `json:"from"`
	To      int64   `json:"to"`
	Hours   float64 `json:"hours"`
	DropPct float64 `json:"dropPct"`
}

// Rate is the average loss over a window of days.
type Rate struct {
	Days       int     `json:"days"`
	Hours      float64 `json:"hours"` // parked time it is based on
	PctPerDay  float64 `json:"pctPerDay"`
	Enough     bool    `json:"enough"`
	DropPctSum float64 `json:"dropPct"`
}

type Report struct {
	Stretches []Stretch `json:"stretches"` // newest first
	Week      Rate      `json:"week"`
	Month     Rate      `json:"month"`
}

// Stretches finds the parked stays in readings that are in time order.
func Stretches(pts []store.TimePoint) []Stretch {
	var out []Stretch
	var run []store.TimePoint
	flush := func() {
		if len(run) >= 2 {
			a, b := run[0], run[len(run)-1]
			hours := float64(b.TS-a.TS) / 3600
			drop := *a.SoC - *b.SoC
			if hours >= minStretchHours && drop <= maxDropPct {
				out = append(out, Stretch{From: a.TS, To: b.TS, Hours: hours, DropPct: max(0, drop)})
			}
		}
		run = nil
	}
	for _, p := range pts {
		if p.Driving || p.Charging || p.SoC == nil {
			flush()
			continue
		}
		if n := len(run); n > 0 {
			last := run[n-1]
			drove := last.Odo != nil && p.Odo != nil && *p.Odo-*last.Odo > driveKm
			charged := *p.SoC-*last.SoC > chargeJumpPct
			if drove || charged {
				flush()
			}
		}
		run = append(run, p)
	}
	flush()
	return out
}

func rate(all []Stretch, now int64, days int) Rate {
	r := Rate{Days: days}
	from := now - int64(days)*86400
	for _, s := range all {
		if s.From >= from {
			r.Hours += s.Hours
			r.DropPctSum += s.DropPct
		}
	}
	if r.Hours >= minTotalHours {
		r.Enough = true
		r.PctPerDay = r.DropPctSum / r.Hours * 24
	}
	return r
}

// Analyze works out the weekly and monthly rates. pts must cover at least the last 30 days.
func Analyze(pts []store.TimePoint, now int64) Report {
	all := Stretches(pts)
	rep := Report{Week: rate(all, now, 7), Month: rate(all, now, 30), Stretches: []Stretch{}}
	for i := len(all) - 1; i >= 0 && len(rep.Stretches) < 8; i-- {
		rep.Stretches = append(rep.Stretches, all[i])
	}
	return rep
}
