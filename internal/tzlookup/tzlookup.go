// Package tzlookup finds the IANA time zone for a coordinate, fully offline
// (the boundary data is compiled in).
package tzlookup

import (
	"sync"
	"time"
	_ "time/tzdata" // zone rules built in, so it works without system tzdata (Windows, minimal images)

	"github.com/ringsaturn/tzf"
)

var (
	once   sync.Once
	finder tzf.F
	ferr   error
)

func load() {
	once.Do(func() { finder, ferr = tzf.NewDefaultFinder() })
}

// Zone returns the IANA name (for example "America/New_York") at a point, or
// "" if it cannot be determined.
func Zone(lat, lon float64) string {
	if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		return ""
	}
	load()
	if ferr != nil || finder == nil {
		return ""
	}
	name := finder.GetTimezoneName(lon, lat)
	if name == "" {
		return ""
	}
	if _, err := time.LoadLocation(name); err != nil {
		return ""
	}
	return name
}

// Location returns the zone at a point, or fallback when unknown.
func Location(lat, lon float64, fallback *time.Location) *time.Location {
	if name := Zone(lat, lon); name != "" {
		if l, err := time.LoadLocation(name); err == nil {
			return l
		}
	}
	return fallback
}
