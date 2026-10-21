package server

import (
	"net/http"
	"sort"
	"time"

	"github.com/ZachCurry13/cartrak/internal/analytics"
	"github.com/ZachCurry13/cartrak/internal/places"
	"github.com/ZachCurry13/cartrak/internal/store"
)

// Wrapped, the glowing map, achievements and the energy flow: the fun side of Insights.

// tripsWithPlaces is every finished trip, as the analysis wants it and as stored (for where each ended).
func (s *Server) tripsWithPlaces(from int64) ([]analytics.Trip, []*store.Trip, error) {
	raw, err := s.S.ListTrips(store.TripFilter{From: from, All: true})
	if err != nil {
		return nil, nil, err
	}
	var done []*store.Trip
	for _, t := range raw {
		if !t.Open && t.EndTS != nil {
			done = append(done, t)
		}
	}
	ts, err := s.analyticsTrips(from)
	return ts, done, err
}

// towns maps each town a trip ended in to when it was first visited, from the towns already looked up (never asking now).
func (s *Server) towns(raw []*store.Trip) map[string]int64 {
	names := s.cityNamer()
	out := map[string]int64{}
	for _, t := range raw {
		if city := names.name(t.EndLat, t.EndLon); city != "" {
			if at, seen := out[city]; !seen || t.StartTS < at {
				out[city] = t.StartTS
			}
		}
	}
	names.lookupLater()
	return out
}

// storedKwh is how much more energy the battery held at to than at from, from the battery level and its size.
func (s *Server) storedKwh(from, to int64) float64 {
	rs, err := s.S.SocReadings(from)
	capacity := s.capacityKwh()
	if err != nil || len(rs) == 0 || capacity == nil {
		return 0
	}
	first, last := rs[0][1], rs[0][1]
	for _, r := range rs {
		if int64(r[0]) <= to {
			last = r[1]
		}
	}
	return (last - first) / 100 * *capacity
}

type namedVisits struct {
	Name   string `json:"name"`
	Visits int    `json:"visits"`
}

func (s *Server) wrapped(w http.ResponseWriter, r *http.Request) {
	loc := s.defaultLoc()
	period := r.URL.Query().Get("period")
	from, to, _, ok2 := analytics.Period(period, loc)
	if !ok2 {
		period = time.Now().In(loc).AddDate(0, -1, 0).Format("2006-01")
		from, to, _, _ = analytics.Period(period, loc)
	}
	ts, raw, err1 := s.tripsWithPlaces(0)
	cs, err2 := s.analyticsCharges(0)
	if err1 != nil || err2 != nil {
		fail(w, 500, "db error")
		return
	}
	var gas analytics.Gas
	if s.Cfg != nil {
		g := s.Cfg.Gas()
		gas = analytics.Gas{LitersPerKm: g.LitersPerKm, PricePerLiter: g.PricePerLiter, GridKgPerKwh: s.Cfg.GridCO2()}
	}
	var inTs []analytics.Trip
	for _, t := range ts {
		if t.Start >= from && t.Start < to {
			inTs = append(inTs, t)
		}
	}
	var inCs []analytics.Charge
	for _, c := range cs {
		if c.Start >= from && c.Start < to {
			inCs = append(inCs, c)
		}
	}
	// towns visited, and the named places visited most, in the period
	towns := s.towns(raw)
	named, _ := s.S.ListNamedPlaces()
	visits := map[string]int{}
	townsHere := map[string]bool{}
	names := s.cityNamer()
	for _, t := range raw {
		if t.StartTS < from || t.StartTS >= to {
			continue
		}
		if n := places.Name(t.EndLat, t.EndLon, named); n != "" && n != "Home" {
			visits[n]++
		}
		if c := names.name(t.EndLat, t.EndLon); c != "" {
			townsHere[c] = true
		}
	}
	var top []namedVisits
	for n, v := range visits {
		top = append(top, namedVisits{n, v})
	}
	sort.Slice(top, func(i, j int) bool {
		return top[i].Visits > top[j].Visits || (top[i].Visits == top[j].Visits && top[i].Name < top[j].Name)
	})
	if len(top) > 3 {
		top = top[:3]
	}
	var earned []analytics.Badge
	for _, b := range analytics.Badges(ts, cs, towns, loc) {
		if b.Earned >= from && b.Earned < to {
			earned = append(earned, b)
		}
	}
	end := min(to, time.Now().Unix())
	// car care: the service visits logged in the period
	type care struct {
		Visits int      `json:"visits"`
		Cost   float64  `json:"cost"`
		Items  []string `json:"items"`
	}
	cc := care{Items: []string{}}
	if vs, err := s.S.Visits(from, to, "logged"); err == nil {
		seenItem := map[string]bool{}
		for _, v := range vs {
			cc.Visits++
			if v.Cost != nil {
				cc.Cost += *v.Cost
			}
			for _, it := range v.Items {
				if !seenItem[it] {
					seenItem[it] = true
					cc.Items = append(cc.Items, it)
				}
			}
		}
	}
	ok(w, map[string]any{
		"service":       cc,
		"recap":         analytics.PeriodRecap(period, ts, cs, gas, loc),
		"personalities": analytics.Personalities(inTs, inCs, loc),
		"flow":          analytics.EnergyFlow(ts, cs, from, to, s.storedKwh(from, end)),
		"ring":          analytics.TypicalDay(inTs, inCs, from, end, loc),
		"moments":       analytics.Records(inTs, inCs, loc),
		"badges":        earned,
		"towns":         len(townsHere),
		"places":        top,
		"currency":      s.currency(),
	})
}
