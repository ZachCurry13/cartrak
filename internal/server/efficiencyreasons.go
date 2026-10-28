package server

import (
	"context"
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
	// Home asks for this on every visit and it takes seconds to work out (a trip at a time, with the weather), so an answer that is
	// already known is given at once, even a few minutes old, and a fresh one is worked out in the background for the next visit. The
	// very first call after a start waits a few seconds at most; after that it answers "not ready yet" and the card stays away until
	// the next visit. The answer is also worked out shortly after CarTrak starts (see ReasonsWarmup), so a visitor rarely meets that.
	s.cc.reasonsMu.Lock()
	out, at, same := s.cc.reasonsOut, s.cc.reasonsAt, s.cc.reasonsDays == days
	s.cc.reasonsMu.Unlock()
	if out != nil && same {
		if time.Since(at) > 5*time.Minute {
			s.refreshReasons(days)
		}
		ok(w, out)
		return
	}
	done := s.refreshReasons(days)
	select {
	case <-done:
		s.cc.reasonsMu.Lock()
		out, same = s.cc.reasonsOut, s.cc.reasonsDays == days
		s.cc.reasonsMu.Unlock()
		if out != nil && same {
			ok(w, out)
			return
		}
		fail(w, 500, "db error")
	case <-time.After(3 * time.Second):
		ok(w, map[string]any{"days": days, "trips": 0, "usedKwh": 0, "usualKwh": 0, "items": []map[string]any{}, "pending": true})
	case <-r.Context().Done():
	}
}

// refreshReasons works the answer out in the background, one at a time, and says when it is done. A call while one is running gets
// the channel of the running one.
func (s *Server) refreshReasons(days int) <-chan struct{} {
	c := &s.cc
	c.reasonsMu.Lock()
	defer c.reasonsMu.Unlock()
	if c.reasonsBusy != nil {
		return c.reasonsBusy
	}
	done := make(chan struct{})
	c.reasonsBusy = done
	go func() {
		fresh, err := s.computeReasons(days)
		c.reasonsMu.Lock()
		c.reasonsBusy = nil
		if err == nil {
			c.reasonsOut, c.reasonsDays, c.reasonsAt = fresh, days, time.Now()
		}
		c.reasonsMu.Unlock()
		close(done)
	}()
	return done
}

// ReasonsWarmup works the answer out once, shortly after CarTrak starts, so the first visit to Home is not the one that waits.
func (s *Server) ReasonsWarmup(ctx context.Context) {
	select {
	case <-time.After(45 * time.Second):
		select {
		case <-s.refreshReasons(60):
		case <-ctx.Done():
		}
	case <-ctx.Done():
	}
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
