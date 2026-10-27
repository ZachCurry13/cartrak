package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

// efficiencyReasons adds up, across recent trips, how much energy each reason cost or saved compared with your usual:
// the speeds you drove, the weather, and everything else. GET /api/efficiency-reasons?days=60
func (s *Server) efficiencyReasons(w http.ResponseWriter, r *http.Request) {
	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if days < 7 || days > 365 {
		days = 60
	}
	// Home asks for this on every visit and it takes seconds to work out, so an answer that is already known is given at once, even a
	// few minutes old, and a fresh one is worked out in the background for the next visit. Only the very first call has to wait.
	s.cc.reasonsMu.Lock()
	out, at, same := s.cc.reasonsOut, s.cc.reasonsAt, s.cc.reasonsDays == days
	refreshing := s.cc.reasonsBusy
	if out != nil && same {
		if time.Since(at) > 5*time.Minute && !refreshing {
			s.cc.reasonsBusy = true
			go func() {
				fresh, err := s.computeReasons(days)
				s.cc.reasonsMu.Lock()
				s.cc.reasonsBusy = false
				if err == nil {
					s.cc.reasonsOut, s.cc.reasonsDays, s.cc.reasonsAt = fresh, days, time.Now()
				}
				s.cc.reasonsMu.Unlock()
			}()
		}
		s.cc.reasonsMu.Unlock()
		ok(w, out)
		return
	}
	s.cc.reasonsMu.Unlock()
	fresh, err := s.computeReasons(days)
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	s.cc.reasonsMu.Lock()
	s.cc.reasonsOut, s.cc.reasonsDays, s.cc.reasonsAt = fresh, days, time.Now()
	s.cc.reasonsMu.Unlock()
	ok(w, fresh)
}

// computeReasons works out where the energy went over the last days: the speeds driven, the weather, and everything else.
func (s *Server) computeReasons(days int) (map[string]any, error) {
	ts, err := s.S.ListTrips(store.TripFilter{From: time.Now().AddDate(0, 0, -days).Unix(), Limit: 400})
	if err != nil {
		return nil, err
	}
	extra := map[string]float64{}
	haveWind := false
	var used, usual float64
	n := 0
	for _, t := range ts {
		if t.Open || t.EndTS == nil || t.DistanceKm < 3 {
			continue
		}
		rep, err := s.whyFor(t)
		if err != nil || !rep.Ready {
			continue
		}
		n++
		km := rep.Facts.Km
		base := rep.UsualKwhPerKm * km // what an average trip of this length would use
		used += rep.KwhPerKm * km
		usual += base
		for _, f := range rep.Factors {
			extra[f.Key] += f.Pct / 100 * base
			if f.Key == "wind" {
				haveWind = true
			}
		}
	}
	items := []map[string]any{}
	for _, k := range []string{"speed", "temperature", "wind", "other"} {
		if k == "wind" && !haveWind {
			continue // the wind lookup is off or had no data
		}
		if usual > 0 {
			items = append(items, map[string]any{"key": k, "kwh": extra[k], "pctOfUse": extra[k] / usual * 100})
		}
	}
	return map[string]any{"days": days, "trips": n, "usedKwh": used, "usualKwh": usual, "items": items}, nil
}
