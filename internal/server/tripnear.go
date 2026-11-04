package server

import (
	"github.com/ZachCurry13/cartrak/internal/superchargers"
	"github.com/ZachCurry13/cartrak/internal/tripchain"
)

const nearbyM = 15000.0 // a charger within this of a hotel is worth naming

// addNearby names, for each night spent somewhere that is not a charger (a hotel), the closest charger from the lists in use and
// how far it is, so the evening or the next morning can start there.
func (s *Server) addNearby(days []tripchain.Day, lists [][]superchargers.Site, avoid map[string]bool) {
	var all []superchargers.Site
	for _, l := range lists {
		for _, u := range l {
			if !avoid[chargerKey(u.Lat, u.Lon)] {
				all = append(all, u)
			}
		}
	}
	for i := range days {
		for j := range days[i].Stops {
			st := &days[i].Stops[j]
			if st.Kind != "overnight" || st.Kw > 0 {
				continue
			}
			if site, ok := superchargers.Nearest(all, st.Lat, st.Lon, nearbyM); ok {
				name := site.Name
				if site.Net == "" {
					name = "Supercharger: " + name
				}
				st.Near = &tripchain.Near{Station: tripchain.Station{Name: name, Lat: site.Lat, Lon: site.Lon, Kw: site.Kw, PS: site.PS, Net: site.Net}, Km: superchargers.DistKm(st.Lat, st.Lon, site.Lat, site.Lon)}
			}
		}
	}
}
