package server

import (
	"net/http"
	"time"

	"github.com/ZachCurry13/cartrak/internal/periodstats"
	"github.com/ZachCurry13/cartrak/internal/places"
)

// periodStats is distance, energy, cost and time as bars by day, month or year, plus charging split into home, work and
// public. GET /api/stats?period=day&at=2026-10 (at is a month for day bars, a year for month bars, ignored for year bars)
func (s *Server) periodStats(w http.ResponseWriter, r *http.Request) {
	loc := s.defaultLoc()
	period := r.URL.Query().Get("period")
	at := time.Now().In(loc)
	if v := r.URL.Query().Get("at"); v != "" {
		layout := "2006-01"
		if len(v) == 4 {
			layout = "2006"
		}
		t, err := time.ParseInLocation(layout, v, loc)
		if err != nil {
			fail(w, 400, "at must look like 2026-10 or 2026")
			return
		}
		at = t
	}
	from, to, valid := periodstats.Range(period, at, s.S.FirstSampleTS(), time.Now())
	if !valid {
		fail(w, 400, "period must be day, month or year")
		return
	}
	pt, err := s.S.PeriodTrips(from.Unix(), to.Unix())
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	sessions, err := s.chargeSessions(5000)
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	named, _ := s.S.ListNamedPlaces()
	var trips []periodstats.Trip
	for _, t := range pt {
		trips = append(trips, periodstats.Trip{Start: t.Start, DistanceKm: t.DistanceKm, Seconds: t.End - t.Start})
	}
	var charges []periodstats.Charge
	for _, c := range sessions {
		if c.AddedKwh == nil || c.StartTS < from.Unix() || c.StartTS >= to.Unix() {
			continue
		}
		x := periodstats.Charge{Start: c.StartTS, AddedKwh: *c.AddedKwh, Kind: periodstats.KindOf(c.Where, places.Name(c.Lat, c.Lon, named))}
		if c.Cost != nil {
			x.Cost = *c.Cost
		}
		if c.EndTS != nil {
			x.Seconds = *c.EndTS - c.StartTS
		}
		charges = append(charges, x)
	}
	ok(w, map[string]any{"stats": periodstats.Build(period, from, to, trips, charges), "currency": s.currency(), "at": at.Format("2006-01")})
}
