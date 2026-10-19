// Package windcalc works out, from where a trip went and the hourly wind along the way, how much of the wind was
// against the car, how much was sideways, and roughly what that did to the energy used.
package windcalc

import (
	"math"

	"github.com/ZachCurry13/cartrak/internal/weather"
)

// Point is one stored reading of a moving car.
type Point struct {
	TS       int64
	Lat, Lon float64
	Speed    float64 // km/h
}

// Lookup gives the wind at a place and time, or false when it is not known.
type Lookup func(lat, lon float64, ts int64) (weather.Hour, bool)

// Stats is the wind over a trip, weighted by distance.
type Stats struct {
	HeadKph   float64 `json:"headKph"`   // average wind along the road: positive is a headwind, negative a tailwind
	CrossKph  float64 `json:"crossKph"`  // average sideways wind
	GustKph   float64 `json:"gustKph"`   // the strongest gust met
	EffectPct float64 `json:"effectPct"` // rough change in energy used because of the wind: positive costs
	Km        float64 `json:"km"`        // the distance this rests on
}

const (
	minSpeed = 30.0 // below this, wind hardly matters for the energy used
	maxGap   = 300  // seconds
)

func rad(d float64) float64 { return d * math.Pi / 180 }

func distanceKm(lat1, lon1, lat2, lon2 float64) float64 {
	dLat, dLon := rad(lat2-lat1), rad(lon2-lon1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(rad(lat1))*math.Cos(rad(lat2))*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 6371 * 2 * math.Asin(math.Sqrt(a))
}

// bearing is the compass direction from one point to the next, in degrees.
func bearing(lat1, lon1, lat2, lon2 float64) float64 {
	y := math.Sin(rad(lon2-lon1)) * math.Cos(rad(lat2))
	x := math.Cos(rad(lat1))*math.Sin(rad(lat2)) - math.Sin(rad(lat1))*math.Cos(rad(lat2))*math.Cos(rad(lon2-lon1))
	return math.Mod(math.Atan2(y, x)*180/math.Pi+360, 360)
}

// dragShare is the part of the energy used at a speed that goes into pushing through the air.
func dragShare(kph float64) float64 { return math.Min(0.7, math.Max(0.1, (kph-20)/100)) }

// Estimate pairs each reading with the one before it and adds up the wind met on each stretch. It returns false when
// no stretch had a wind reading.
func Estimate(points []Point, look Lookup) (Stats, bool) {
	var s Stats
	var effKm float64
	for i := 1; i < len(points); i++ {
		a, b := points[i-1], points[i]
		dt := b.TS - a.TS
		if dt <= 0 || dt > maxGap || a.Speed < minSpeed || b.Speed < minSpeed {
			continue
		}
		km := distanceKm(a.Lat, a.Lon, b.Lat, b.Lon)
		if km < 0.05 || km > float64(dt)/3600*250 {
			continue
		}
		w, ok := look((a.Lat+b.Lat)/2, (a.Lon+b.Lon)/2, (a.TS+b.TS)/2)
		if !ok {
			continue
		}
		theta := rad(bearing(a.Lat, a.Lon, b.Lat, b.Lon) - w.DirDeg) // angle between where the car goes and where the wind comes from
		head, cross := w.SpeedKm*math.Cos(theta), math.Abs(w.SpeedKm*math.Sin(theta))
		v := (a.Speed + b.Speed) / 2
		air := math.Max(0.3*v, v+head)
		s.HeadKph += head * km
		s.CrossKph += cross * km
		s.EffectPct += dragShare(v) * (air*air/(v*v) - 1) * 100 * km
		effKm += km
		if w.GustKm > s.GustKph {
			s.GustKph = w.GustKm
		}
	}
	if effKm < 3 {
		return Stats{}, false
	}
	s.HeadKph /= effKm
	s.CrossKph /= effKm
	s.EffectPct /= effKm
	s.Km = effKm
	return s, true
}
