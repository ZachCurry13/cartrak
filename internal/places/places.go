// Package places matches a position to the places the owner has named.
package places

import (
	"math"

	"github.com/ZachCurry13/cartrak/internal/store"
)

func meters(lat1, lon1, lat2, lon2 float64) float64 {
	const r = 6371000.0
	a1, a2 := lat1*math.Pi/180, lat2*math.Pi/180
	dLat, dLon := (lat2-lat1)*math.Pi/180, (lon2-lon1)*math.Pi/180
	h := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(a1)*math.Cos(a2)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 2 * r * math.Asin(math.Sqrt(h))
}

// Match is the named place the position is inside, the closest one when more than one overlaps; nil when none.
func Match(lat, lon float64, all []*store.NamedPlace) *store.NamedPlace {
	var best *store.NamedPlace
	bestD := math.MaxFloat64
	for _, p := range all {
		if d := meters(lat, lon, p.Lat, p.Lon); d <= p.RadiusM && d < bestD {
			best, bestD = p, d
		}
	}
	return best
}

// Name is the match's name, or "" for none or an unknown position.
func Name(lat, lon *float64, all []*store.NamedPlace) string {
	if lat == nil || lon == nil {
		return ""
	}
	if p := Match(*lat, *lon, all); p != nil {
		return p.Name
	}
	return ""
}

// Distance is how far the position is from the place's center, in meters.
func Distance(lat, lon float64, p *store.NamedPlace) float64 { return meters(lat, lon, p.Lat, p.Lon) }
