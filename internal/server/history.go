package server

import (
	"net/http"
	"time"

	"github.com/ZachCurry13/cartrak/internal/history"
	"github.com/ZachCurry13/cartrak/internal/store"
)

// monthRange is the first moment of the month asked for (2026-10, or this month) and of the one after it.
func (s *Server) monthRange(month string) (time.Time, time.Time, bool) {
	loc := s.defaultLoc()
	t := time.Now().In(loc)
	if month != "" {
		p, err := time.ParseInLocation("2006-01", month, loc)
		if err != nil {
			return t, t, false
		}
		t = p
	}
	from := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, loc)
	return from, from.AddDate(0, 1, 0), true
}

// historyFeed is a month of driving, charging and parked time as one feed, with the month's totals.
// GET /api/history?month=2026-10
func (s *Server) historyFeed(w http.ResponseWriter, r *http.Request) {
	from, to, valid := s.monthRange(r.URL.Query().Get("month"))
	if !valid {
		fail(w, 400, "the month must look like 2026-10")
		return
	}
	trips, err := s.S.ListTrips(store.TripFilter{From: from.Unix(), To: to.Unix(), Limit: 1000})
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	sessions, err := s.chargeSessions(2000)
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	var charges []history.Charge
	for _, c := range sessions {
		if c.StartTS < from.Unix() || c.StartTS >= to.Unix() {
			continue
		}
		charges = append(charges, history.Charge{ID: c.ID, Start: c.StartTS, End: c.EndTS, StartSoC: c.StartSoC, EndSoC: c.EndSoC,
			Lat: c.Lat, Lon: c.Lon, AddedKwh: c.AddedKwh, Cost: c.Cost, Where: c.Where})
	}
	named, _ := s.S.ListNamedPlaces()
	now := time.Now().Unix()
	events := history.Build(trips, charges, named, from.Unix(), to.Unix(), now)
	s.placeEvents(events, trips, named)
	ok(w, map[string]any{"month": from.Format("2006-01"), "from": from.Unix(), "to": to.Unix(), "events": events,
		"totals": history.Sum(events, from.Unix(), to.Unix(), now), "tz": s.defaultLoc().String(), "currency": s.currency()})
}
