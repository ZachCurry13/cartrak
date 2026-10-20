package server

import (
	"github.com/ZachCurry13/cartrak/internal/charging"
	"github.com/ZachCurry13/cartrak/internal/history"
	"github.com/ZachCurry13/cartrak/internal/store"
)

func (d *daySummary) addCharge(c *charging.Session) {
	d.Sessions++
	if c.Station != "" && len(c.Station) >= 2 && c.Station[:2] == "DC" || c.PeakKw >= 30 {
		d.FastSessions++
	}
	if c.PeakKw > d.PeakKw {
		d.PeakKw = c.PeakKw
	}
	if c.EndTS != nil {
		d.ChargeSec += *c.EndTS - c.StartTS
	}
	switch {
	case c.Cost == nil:
		d.Unpriced++
	case c.CostActual:
		d.Receipts++
	default:
		d.Estimated++
	}
	if c.AddedKwh != nil {
		d.addedKwh += *c.AddedKwh
	}
	if c.Cost != nil {
		d.cost += *c.Cost
	}
}

func (d *daySummary) finish(list []*store.Trip, events []history.Event, dayEnd int64) {
	var km float64
	var sec int64
	for _, t := range list {
		if t.Open || t.EndTS == nil {
			continue
		}
		d.Trips++
		km += t.DistanceKm
		sec += *t.EndTS - t.StartTS
		if t.MaxSpeedKph > d.TopKph {
			d.TopKph = t.MaxSpeedKph
		}
		if d.FirstStart == 0 || t.StartTS < d.FirstStart {
			d.FirstStart = t.StartTS
		}
		if *t.EndTS > d.LastEnd {
			d.LastEnd = *t.EndTS
		}
	}
	if sec > 0 {
		d.AvgMovingKph = km / (float64(sec) / 3600)
	}
	if d.ChargeSec > 0 {
		d.AvgKw = d.addedKwh / (float64(d.ChargeSec) / 3600)
	}
	if km > 0 && d.cost > 0 {
		d.CostPerKm = d.cost / km
	}
	for _, e := range events {
		if e.Kind != "parked" {
			continue
		}
		end := dayEnd
		if e.End != nil {
			end = *e.End
		}
		secs := end - e.Start
		if secs < dayStopMinutes*60 {
			continue
		}
		d.Stops++
		d.StoppedSec += secs
		if secs > d.LongestStop {
			d.LongestStop, d.LongestAt = secs, e.Place
		}
	}
	if len(events) > 0 {
		d.StartSoC, d.EndSoC = events[0].StartSoC, events[len(events)-1].EndSoC
		if d.EndSoC == nil {
			for i := len(events) - 1; i >= 0; i-- {
				if events[i].EndSoC != nil {
					d.EndSoC = events[i].EndSoC
					break
				}
			}
		}
		if d.StartSoC == nil {
			for _, e := range events {
				if e.StartSoC != nil {
					d.StartSoC = e.StartSoC
					break
				}
			}
		}
	}
}
