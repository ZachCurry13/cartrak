package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/ZachCurry13/cartrak/internal/charging"
)

// chargeForecast prices a charge from one battery percentage to another at the owner's prices.
// GET /api/charge-forecast?from=30&to=80 (from defaults to the car's level now, to to 80).
func (s *Server) chargeForecast(w http.ResponseWriter, r *http.Request) {
	capacity := s.capacityKwh()
	var now *float64
	if x, _ := s.S.LatestSample(); x != nil {
		now = x.SoCPct
	}
	num := func(key string, def *float64) (float64, bool) {
		if v := r.URL.Query().Get(key); v != "" {
			f, err := strconv.ParseFloat(v, 64)
			return f, err == nil
		}
		if def != nil {
			return *def, true
		}
		return 0, false
	}
	eighty := 80.0
	from, okFrom := num("from", now)
	to, okTo := num("to", &eighty)
	out := map[string]any{"currency": s.currency(), "socNow": now, "capacityKwh": capacity, "from": from, "to": to}
	if capacity == nil || !okFrom || !okTo {
		out["ok"] = false
		ok(w, out)
		return
	}
	acEff := 0.9
	if s.Cfg != nil {
		acEff = s.Cfg.ChargeEfficiency()
	}
	if _, eff, err := s.chargeSessionsEff(1000); err == nil {
		acEff = eff.Value
	}
	f, valid := charging.ForecastCharge(*capacity, from, to, s.prices(), acEff)
	out["ok"], out["forecast"], out["efficiency"] = valid, f, acEff
	ok(w, out)
}

// monthSoFar is this month's charging bill to date and where it is heading, when the first month in the list is
// the current one and its sessions are all priced.
func (s *Server) monthSoFar(months []monthOut, loc *time.Location, now time.Time) map[string]any {
	if len(months) == 0 || months[0].Month.Month != now.In(loc).Format("2006-01") || months[0].Month.Priced == 0 {
		return nil
	}
	local := now.In(loc)
	days := time.Date(local.Year(), local.Month()+1, 0, 0, 0, 0, 0, loc).Day()
	day := float64(local.Day()-1) + float64(local.Hour())/24
	out := map[string]any{"cost": months[0].Month.Cost, "kwh": months[0].Month.Kwh, "daysLeft": days - local.Day()}
	if p, ok := charging.MonthProjection(months[0].Month.Cost, day, days); ok {
		out["projected"] = p
	}
	return out
}
