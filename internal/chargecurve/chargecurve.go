// Package chargecurve turns the readings of one charging session into a charging curve: power against battery
// level, with the numbers worth comparing between chargers.
package chargecurve

import "sort"

// Point is one reading during a session.
type Point struct {
	TS  int64   `json:"ts"`
	SoC float64 `json:"soc"`
	Kw  float64 `json:"kw"`
}

// Curve is the points and what they add up to.
type Curve struct {
	Points        []Point  `json:"points"`
	PeakKw        float64  `json:"peakKw"`
	AvgKw         *float64 `json:"avgKw"`         // energy added over the time it took
	AddedKwh      *float64 `json:"addedKwh"`      // as the car measured it
	Minutes       float64  `json:"minutes"`       // length of the session
	Minutes10to80 *float64 `json:"minutes10to80"` // only when the session covers both levels
	At80Kw        *float64 `json:"at80Kw"`        // power as it passed about 80 percent
}

// Build summarizes points (any order). energyStart and energyEnd are the energy left in the battery at the start and
// the end of the session, when the car reported it.
func Build(points []Point, energyStart, energyEnd *float64) Curve {
	ps := append([]Point(nil), points...)
	sort.Slice(ps, func(i, j int) bool { return ps[i].TS < ps[j].TS })
	c := Curve{Points: ps}
	if len(ps) == 0 {
		c.Points = []Point{}
		return c
	}
	c.Minutes = float64(ps[len(ps)-1].TS-ps[0].TS) / 60
	var sum float64
	var n int
	for _, p := range ps {
		if p.Kw > c.PeakKw {
			c.PeakKw = p.Kw
		}
		if p.Kw > 0.2 {
			sum += p.Kw
			n++
		}
	}
	if energyStart != nil && energyEnd != nil && *energyEnd > *energyStart {
		added := *energyEnd - *energyStart
		c.AddedKwh = &added
		if c.Minutes >= 2 {
			avg := added / (c.Minutes / 60)
			c.AvgKw = &avg
		}
	}
	if c.AvgKw == nil && n > 0 {
		avg := sum / float64(n)
		c.AvgKw = &avg
	}
	first := func(level float64) *Point {
		for i := range ps {
			if ps[i].SoC >= level {
				return &ps[i]
			}
		}
		return nil
	}
	a, b := first(10), first(80)
	if a != nil && b != nil && ps[0].SoC <= 11 && b.TS > a.TS {
		m := float64(b.TS-a.TS) / 60
		c.Minutes10to80 = &m
	}
	if b != nil && b.Kw > 0.2 {
		k := b.Kw
		c.At80Kw = &k
	}
	return c
}
