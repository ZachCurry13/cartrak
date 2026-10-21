// Package trips turns a stream of vehicle readings into trips and charge
// sessions, and classifies trips as personal or work.
package trips

import (
	"math"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

const (
	Personal = "personal"
	Work     = "work"
)

// Classify returns the category of the highest-priority enabled rule that
// matches a trip starting at t (already in the user's time zone) at lat/lon.
// ok is false when no rule matches (caller keeps the default).
func Classify(rules []store.Rule, t time.Time, lat, lon *float64) (cat string, ruleID int64, ok bool) {
	min := t.Hour()*60 + t.Minute()
	bit := 1 << uint(t.Weekday()) // Sunday = 1
	best := -1 << 31
	for _, r := range rules {
		if !r.Enabled || r.Days&bit == 0 || r.Priority < best {
			continue
		}
		if !inWindow(min, r.StartMin, r.EndMin) || !inGeo(r, lat, lon) {
			continue
		}
		if r.Priority > best || !ok {
			cat, ruleID, ok, best = r.Category, r.ID, true, r.Priority
		}
	}
	return
}

// inWindow handles windows that wrap past midnight (start > end).
func inWindow(min, start, end int) bool {
	if start <= end {
		return min >= start && min < end
	}
	return min >= start || min < end
}

func inGeo(r store.Rule, lat, lon *float64) bool {
	if r.GeoLat == nil || r.GeoLon == nil || r.GeoRadius == nil {
		return true
	}
	if lat == nil || lon == nil {
		return false
	}
	return Haversine(*lat, *lon, *r.GeoLat, *r.GeoLon) <= *r.GeoRadius
}

// Haversine returns the distance in meters between two points.
func Haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371000.0
	rad := math.Pi / 180
	dLat, dLon := (lat2-lat1)*rad, (lon2-lon1)*rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 2 * R * math.Asin(math.Sqrt(a))
}
