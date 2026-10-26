package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/insights"
	"github.com/ZachCurry13/cartrak/internal/speedeff"
	"github.com/ZachCurry13/cartrak/internal/store"
	"github.com/ZachCurry13/cartrak/internal/trips"
	"github.com/ZachCurry13/cartrak/internal/tripwhy"
)

// whyBaseline is what the driver's own history says about speed and temperature, kept for half an hour.
func (s *Server) whyBaseline() tripwhy.Baseline {
	s.cc.whyMu.Lock()
	defer s.cc.whyMu.Unlock()
	if time.Since(s.cc.whyAt) < 30*time.Minute && s.cc.whyBase.Overall > 0 {
		return s.cc.whyBase
	}
	var b tripwhy.Baseline
	b.TempKmPKwh = map[string]float64{}
	if rows, err := s.S.DrivingReadings(time.Now().AddDate(0, 0, -90).Unix()); err == nil {
		rs := make([]speedeff.Reading, 0, len(rows))
		for _, x := range rows {
			rs = append(rs, speedeff.Reading{TS: int64(x[0]), Odo: x[1], Kwh: x[2], Speed: x[3]})
		}
		var km, kwh float64
		for _, o := range speedeff.Build(rs) {
			b.Bands = append(b.Bands, tripwhy.Band{LoKph: o.LoKph, HiKph: o.HiKph, KmPerKwh: o.KmPerKwh})
			km += o.Km
			kwh += o.Kwh
		}
		if kwh > 0 {
			b.Overall = km / kwh
		}
	}
	from := time.Now().AddDate(0, 0, -366).Unix()
	if ts, err := s.S.EnergyTrips(from); err == nil {
		if temps, err := s.S.TripTemps(from); err == nil {
			rep := insights.Build(ts, s.capacityKwh(), temps, s.defaultLoc())
			var km, kwh float64
			for _, bd := range rep.Bands {
				b.TempKmPKwh[bd.Name] = bd.KmPerKwh
				km += bd.Km
				kwh += bd.Km / bd.KmPerKwh
			}
			if kwh > 0 {
				b.TempOverall = km / kwh
			}
		}
	}
	s.cc.whyBase, s.cc.whyAt = b, time.Now()
	return b
}

// whyFor explains one finished trip from its stored readings.
func (s *Server) whyFor(t *store.Trip) (tripwhy.Report, error) {
	e, has := trips.EnergyUsed(t, s.capacityKwh())
	rows, err := s.S.TripReadings(t.StartTS, *t.EndTS)
	if err != nil {
		return tripwhy.Report{}, err
	}
	rs := make([]tripwhy.Reading, 0, len(rows))
	var wpts []windPoint
	var tsum float64
	tn := 0
	for _, x := range rows {
		if x.Lat != nil && x.Lon != nil && x.Speed != nil {
			wpts = append(wpts, windPoint{ts: x.TS, lat: *x.Lat, lon: *x.Lon, speed: *x.Speed})
		}
		if x.Odo == nil || x.Kwh == nil || x.Speed == nil {
			continue
		}
		rs = append(rs, tripwhy.Reading{TS: x.TS, Odo: *x.Odo, Kwh: *x.Kwh, Speed: *x.Speed, OutsideC: x.Temp})
		if x.Temp != nil {
			tsum += *x.Temp
			tn++
		}
	}
	group := ""
	if tn > 0 {
		group = insights.BandName(tsum / float64(tn))
	}
	kwh := 0.0
	if has {
		kwh = e.Kwh
	}
	// The car keeps its own count of the range braking gave back since the ignition came on; its last value in the trip
	// is the trip's total. (Its other regen counter is documented as a percentage but reads like a different unit.)
	var regen *float64
	if x, _ := s.S.SampleAt(*t.EndTS); x != nil && x.TS >= t.StartTS {
		f := map[string]any{}
		if json.Unmarshal(x.Extra, &f) == nil {
			regen = ford.Number(f, "tripXevBatteryRangeRegenerated")
		}
	}
	var wind *tripwhy.Wind
	if ws, ok := s.windFor(wpts); ok {
		wind = &tripwhy.Wind{HeadKph: ws.HeadKph, CrossKph: ws.CrossKph, GustKph: ws.GustKph, EffectPct: ws.EffectPct}
	}
	rep := tripwhy.BuildWithWind(rs, kwh, s.whyBaseline(), group, regen, wind)
	s.withSmoothness(&rep, t)
	return rep, nil
}

// tripWhy explains one trip's efficiency: speed, temperature, stopping and what is left over.
// GET /api/trips/{id}/why
func (s *Server) tripWhy(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	t, err := s.S.GetTrip(id)
	if err != nil || t == nil || t.Open || t.EndTS == nil {
		fail(w, 404, "no such finished trip")
		return
	}
	rep, err := s.whyFor(t)
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, rep)
}

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
