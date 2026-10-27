package server

import (
	"net/http"
	"time"

	"github.com/ZachCurry13/cartrak/internal/planahead"
	"github.com/ZachCurry13/cartrak/internal/store"
)

// planAhead forecasts the next week of driving from the last eight weeks and says when the battery would need a charge.
func (s *Server) planAhead(w http.ResponseWriter, r *http.Request) {
	loc := s.defaultLoc()
	now := time.Now().In(loc)
	midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	start := midnight.AddDate(0, 0, -56)
	if first := s.S.FirstSampleTS(); first > 0 {
		f := time.Unix(first, 0).In(loc)
		if fd := time.Date(f.Year(), f.Month(), f.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, 1); fd.After(start) {
			start = fd // the first day is partial, so history starts the day after
		}
	}
	x, _ := s.S.LatestSample()
	capacity := s.capacityKwh()
	if s.fuelCar() { // a gas or hybrid car has no battery to plan around: no "plug in tonight"
		ok(w, map[string]any{"ready": false})
		return
	}
	if x == nil || x.SoCPct == nil || capacity == nil {
		ok(w, map[string]any{"ready": false})
		return
	}
	ts, err := s.S.ListTrips(store.TripFilter{From: start.Unix(), Limit: 5000})
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	perDay := map[string]float64{}
	for _, t := range ts {
		if !t.Open {
			perDay[time.Unix(t.StartTS, 0).In(loc).Format("2006-01-02")] += t.DistanceKm
		}
	}
	var history []planahead.DayKm
	for d := start; d.Before(midnight); d = d.AddDate(0, 0, 1) {
		k := d.Format("2006-01-02")
		history = append(history, planahead.DayKm{Date: k, Km: perDay[k]})
	}
	var kwh, km float64
	for _, t := range s.tripsWithEnergy(ts) {
		if t.EnergyKwh != nil && t.DistanceKm > 1 {
			kwh += *t.EnergyKwh
			km += t.DistanceKm
		}
	}
	reserve := 20.0
	if s.Cfg != nil {
		if l := s.Cfg.LowBatteryLevel(); l > 0 {
			reserve = l
		}
	}
	if km <= 0 {
		ok(w, map[string]any{"ready": false})
		return
	}
	in := planahead.Input{History: history, Today: midnight, KwhPerKm: kwh / km, CapacityKwh: *capacity, StartPct: *x.SoCPct, ReservePct: reserve}
	away := ""
	if b := s.baseFinder(nil).forDay(midnight.Format("2006-01-02")); b != nil { // on a trip away: forecast from the trip so far
		var trip []planahead.DayKm
		for _, h := range history {
			if h.Date >= b.Stretch.From {
				trip = append(trip, h)
			}
		}
		if last, err := time.ParseInLocation("2006-01-02", b.Stretch.To, loc); err == nil && len(trip) >= 2 {
			in.History, in.Away, in.DaysLeft = trip, true, int(last.Sub(midnight).Hours()/24+0.5)
			away = b.Stretch.Label
			if away == "" {
				away = "this trip"
			}
		}
	}
	f, good := planahead.Build(in)
	if !good {
		ok(w, map[string]any{"ready": false, "historyDays": len(history), "needDays": planahead.MinHistoryDays})
		return
	}
	ok(w, map[string]any{"ready": true, "forecast": f, "kwhPerKm": kwh / km, "away": away})
}
