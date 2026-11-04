package server

import (
	"encoding/json"
	"sync"

	"github.com/ZachCurry13/cartrak/internal/superchargers"
)

const chargerNameM = 200.0 // a charge this close to a known charger was at it

var chargerKept struct {
	mu    sync.Mutex
	stamp string
	sites []superchargers.Site
}

// knownChargers are the Supercharger and Electrify America sites CarTrak already holds (from the long-trip planner). Nothing is
// downloaded here: until a list has been fetched there are none.
func (s *Server) knownChargers() []superchargers.Site {
	stamp := s.S.GetKV(superAtKey) + "|" + s.S.GetKV(eaAtKey)
	chargerKept.mu.Lock()
	defer chargerKept.mu.Unlock()
	if stamp == chargerKept.stamp {
		return chargerKept.sites
	}
	var all []superchargers.Site
	for _, k := range []string{superKey, eaKey} {
		var l []superchargers.Site
		if b := s.S.GetKV(k); b != "" && json.Unmarshal([]byte(b), &l) == nil {
			all = append(all, l...)
		}
	}
	chargerKept.stamp, chargerKept.sites = stamp, all
	return all
}

// chargerAt is the name of the known charger at a point ("Supercharger: Mackinaw City, MI"), or "".
func (s *Server) chargerAt(lat, lon *float64) string {
	if lat == nil || lon == nil {
		return ""
	}
	if site, ok := superchargers.Nearest(s.knownChargers(), *lat, *lon, chargerNameM); ok {
		if site.Net != "" {
			return site.Name
		}
		return "Supercharger: " + site.Name
	}
	return ""
}
