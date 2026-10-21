package server

import (
	"net/http"
	"time"

	"github.com/ZachCurry13/cartrak/internal/analytics"
	"github.com/ZachCurry13/cartrak/internal/places"
	"github.com/ZachCurry13/cartrak/internal/store"
)

const coachDays = 60

// chargingTips is the charging coach: a few tips from the last two months of charging, each only when the numbers back it up.
// GET /api/analytics/coach
func (s *Server) chargingTips(w http.ResponseWriter, r *http.Request) {
	now := time.Now().Unix()
	from := now - coachDays*86400
	sessions, err := s.chargeSessions(2000)
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	trips, _ := s.S.ListTrips(store.TripFilter{From: from, All: true})
	named, _ := s.S.ListNamedPlaces()
	pr := s.prices()
	starts := make([]int64, len(sessions))
	for i, c := range sessions {
		starts[i] = c.StartTS
	}
	var cs []analytics.CoachCharge
	for _, c := range sessions {
		if c.Open || c.EndTS == nil || c.StartTS < from || c.AddedKwh == nil {
			continue
		}
		x := analytics.CoachCharge{Start: c.StartTS, End: *c.EndTS, Kwh: *c.AddedKwh, Cost: c.Cost, Where: c.Where, Place: c.Place, PeakKw: c.PeakKw, StartSoC: c.StartSoC, EndSoC: c.EndSoC}
		if c.WallKwh != nil {
			x.WallKwh = *c.WallKwh
		}
		if x.Place == "" && c.Where != "home" {
			x.Place = places.Name(c.Lat, c.Lon, named)
		}
		if pr.Window != nil && pr.Loc != nil {
			x.CheapShare = pr.Window.Share(c.StartTS, *c.EndTS, pr.Loc)
		}
		x.NextUsePct = usedUntilNextCharge(*c.EndTS, starts, trips)
		cs = append(cs, x)
	}
	p := analytics.Prices{Currency: s.currency()}
	if pr.Home != nil {
		p.Home = *pr.Home
	}
	if pr.OffPeak != nil {
		p.Cheap = *pr.OffPeak
	}
	if pr.Public != nil {
		p.Public = *pr.Public
	}
	ok(w, map[string]any{"tips": analytics.Coach(cs, p, coachDays), "days": coachDays, "sessions": len(cs)})
}

// usedUntilNextCharge is how much of the battery the drives after a charge used before the next charge began, in percent.
func usedUntilNextCharge(end int64, starts []int64, trips []*store.Trip) *float64 {
	next := int64(0)
	for _, st := range starts {
		if st > end && (next == 0 || st < next) {
			next = st
		}
	}
	if next == 0 {
		return nil // no charge after it yet: the driving is not over
	}
	var used float64
	seen := false
	for _, t := range trips {
		if t.StartTS < end || t.StartTS >= next || t.StartSoC == nil || t.EndSoC == nil {
			continue
		}
		used += *t.StartSoC - *t.EndSoC
		seen = true
	}
	if !seen {
		return nil
	}
	return &used
}
