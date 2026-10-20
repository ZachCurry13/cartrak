package server

import (
	"net/http"
	"time"

	"github.com/ZachCurry13/cartrak/internal/datacheck"
)

const dataCheckDays = 90

// dataCheck lists things in the last 90 days of recorded data that look wrong: gaps in the odometer, an odometer
// that went backwards, overlapping or impossible trips. It only reads.
func (s *Server) dataCheck(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	from := now.AddDate(0, 0, -dataCheckDays).Unix()
	pts, err := s.S.OdoBetween(from, now.Unix()+1)
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	ts, err := s.S.CheckTrips(from)
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	odo := make([]datacheck.Odo, 0, len(pts))
	for _, p := range pts {
		odo = append(odo, datacheck.Odo{TS: p.TS, Km: p.Km})
	}
	trips := make([]datacheck.Trip, 0, len(ts))
	for _, t := range ts {
		x := datacheck.Trip{ID: t.ID, StartTS: t.StartTS, DistanceKm: t.DistanceKm, Open: t.Open}
		if t.EndTS != nil {
			x.EndTS = *t.EndTS
		}
		trips = append(trips, x)
	}
	f := datacheck.Check(odo, trips, metricReq(r))
	if f == nil {
		f = []datacheck.Finding{}
	}
	ok(w, map[string]any{"days": dataCheckDays, "findings": f, "readings": len(odo), "trips": len(trips)})
}
