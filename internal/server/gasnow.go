package server

import (
	"net/http"
	"time"
)

// gasNow: GET /api/gasnow. What a gallon costs where the car is now, how that moved since last week, and what a distance costs
// in the electric car against the gas car it is compared with. Everything comes from prices already remembered, so it never
// waits on the EIA.
func (s *Server) gasNow(w http.ResponseWriter, r *http.Request) {
	if !s.liveGasOn() {
		ok(w, map[string]any{"live": false})
		return
	}
	var lat, lon *float64
	if x, _ := s.S.LatestSample(); x != nil {
		lat, lon = x.Lat, x.Lon
	}
	if lat == nil {
		if p := s.homePoint(); p != nil {
			lat, lon = &p.Lat, &p.Lon
		}
	}
	now := time.Now()
	q := s.gasQuote(lat, lon, now.Unix())
	if !q.OK {
		ok(w, map[string]any{"live": true, "ready": false})
		return
	}
	out := map[string]any{"live": true, "ready": true, "area": q.Source, "week": q.Week, "perGallon": q.PerGallon, "currency": s.currency()}
	if prev := s.gasQuote(lat, lon, now.AddDate(0, 0, -7).Unix()); prev.OK && prev.Week != q.Week {
		out["change"] = q.PerGallon - prev.PerGallon
	}
	if g := s.Cfg.Gas(); g.Set() {
		out["gasPerKm"] = g.LitersPerKm * q.PerGallon / litersPerGallon
		if p, km := s.avgPricePerKwh(), s.efficiencyFor().KmPerKwh(now.Unix()); p != nil && km > 0 {
			out["electricPerKm"] = *p / km
		}
	}
	ok(w, out)
}
