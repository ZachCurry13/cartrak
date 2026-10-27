package analytics

import (
	"time"
)

// WithEstimates adds, to the trophies that count distance or trips, about when they will be earned at the pace of the last 90 days of
// driving (or of all of it, when it is shorter, once there is a week of it). The odometer trophies go by the odometer's own reading.
func WithEstimates(bs []Badge, ts []Trip, now time.Time) []Badge {
	var latestOdo float64
	cutoff := now.AddDate(0, 0, -90).Unix()
	var km, trips float64
	first := now.Unix()
	for _, t := range ts {
		if t.Km < 0.3 {
			continue
		}
		if t.Start < first {
			first = t.Start
		}
		if t.Start >= cutoff {
			km += t.Km
			trips++
		}
		latestOdo = max(latestOdo, t.OdoKm)
	}
	span := now.Sub(time.Unix(max(first, cutoff), 0)).Hours() / 24
	if span < 7 || km <= 0 {
		return bs
	}
	perDay, tripsPerDay := km/span, trips/span
	out := make([]Badge, len(bs))
	copy(out, bs)
	for i, b := range out {
		if b.Earned > 0 || b.Goal <= 0 {
			continue
		}
		var left, rate float64
		switch b.Key {
		case "km160", "km1600", "km16000", "world":
			left, rate = b.Goal-b.Value, perDay
		case "odo50", "odo100", "odo150":
			left, rate = b.Goal-latestOdo, perDay
		case "trips100", "trips1000":
			left, rate = b.Goal-b.Value, tripsPerDay
		default:
			continue
		}
		if left > 0 && rate > 0 && left/rate < 3650 {
			out[i].Eta = now.Add(time.Duration(left / rate * 24 * float64(time.Hour))).Unix()
		}
	}
	return out
}
