package server

import (
	"net/http"
	"sort"

	"github.com/ZachCurry13/cartrak/internal/places"
	"github.com/ZachCurry13/cartrak/internal/store"
)

type placeStat struct {
	Name   string  `json:"name"`
	Visits int     `json:"visits"`
	Last   int64   `json:"last"`
	Km     float64 `json:"km"` // driven to get there
}

func topPlaces(m map[string]*placeStat, most int) []placeStat {
	out := make([]placeStat, 0, len(m))
	for _, p := range m {
		out = append(out, *p)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Visits > out[j].Visits || (out[i].Visits == out[j].Visits && (out[i].Last > out[j].Last || (out[i].Last == out[j].Last && out[i].Name < out[j].Name)))
	})
	if len(out) > most {
		out = out[:most]
	}
	return out
}

// popularPlaces: GET /api/analytics/places?days=365. The places this car went to most, by the trips that ended there: the places you
// named, and the towns it looked up. Everyone who can see the car can see it.
func (s *Server) popularPlaces(w http.ResponseWriter, r *http.Request) {
	raw, err := s.S.ListTrips(store.TripFilter{From: analyticsFrom(r), All: true})
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	named, _ := s.S.ListNamedPlaces()
	names := s.cityNamer()
	ps, ts := map[string]*placeStat{}, map[string]*placeStat{}
	add := func(m map[string]*placeStat, name string, t *store.Trip) {
		if name == "" {
			return
		}
		p := m[name]
		if p == nil {
			p = &placeStat{Name: name}
			m[name] = p
		}
		p.Visits++
		p.Km += t.DistanceKm
		if t.StartTS > p.Last {
			p.Last = t.StartTS
		}
	}
	trips := 0
	for _, t := range raw {
		if t.Open || t.EndTS == nil || t.DistanceKm < 0.3 {
			continue
		}
		trips++
		add(ps, places.Name(t.EndLat, t.EndLon, named), t)
		add(ts, names.name(t.EndLat, t.EndLon), t)
	}
	names.lookupLater()
	ok(w, map[string]any{"places": topPlaces(ps, 10), "towns": topPlaces(ts, 10), "trips": trips})
}
