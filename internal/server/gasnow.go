package server

import (
	"context"
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

// GasPricesRun keeps the live gas prices current without anyone opening a page: twice a day it fetches again the areas in use whose
// prices are more than 12 hours old (the EIA publishes weekly, on Mondays).
func (s *Server) GasPricesRun(ctx context.Context) {
	tick := time.NewTicker(6 * time.Hour)
	defer tick.Stop()
	first := time.After(time.Minute)
	for {
		select {
		case <-ctx.Done():
			return
		case <-first:
		case <-tick.C:
		}
		if s.liveGasOn() {
			c, cancel := context.WithTimeout(ctx, 2*time.Minute)
			s.gasService().RefreshKnown(c, s.Cfg.EIAKey())
			cancel()
		}
	}
}

// gasHealth is the Status line for live gas prices. It shows only when an EIA key is set.
func (s *Server) gasHealth(now int64) (healthItem, bool) {
	if s.Cfg == nil || s.Cfg.EIAKey() == "" {
		return healthItem{}, false
	}
	const title = "Live gas prices"
	if !s.liveGasOn() {
		return healthItem{"gas", title, "An EIA key is set, but live prices also need city lookup on and US units (More, Settings and Charging settings). Your own gas price is used.", "info", "#/charging-settings", "Open Charging settings"}, true
	}
	st := s.gasService().Status()
	switch {
	case st.Err != "":
		return healthItem{"gas", title, "The last update failed: " + st.Err + ". Your own gas price is used until it works.", "warn", "#/charging-settings", "Open Charging settings"}, true
	case st.OK == 0:
		return healthItem{"gas", title, "Waiting for the first update. It happens when the savings or Fuel pages are opened, and twice a day after that.", "info", "", ""}, true
	}
	return healthItem{"gas", title, "Prices were updated " + ago(st.OK) + ". The EIA publishes a new weekly average each Monday.", "ok", "", ""}, true
}
