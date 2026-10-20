package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/ZachCurry13/cartrak/internal/charging"
)

const kvHomeLat, kvHomeLon = "home_lat", "home_lon"

// homePoint is where the owner said home is (by marking a session as home), nil until then.
func (s *Server) homePoint() *charging.Point {
	la, e1 := strconv.ParseFloat(s.S.GetKV(kvHomeLat), 64)
	lo, e2 := strconv.ParseFloat(s.S.GetKV(kvHomeLon), 64)
	if e1 != nil || e2 != nil {
		return nil
	}
	return &charging.Point{Lat: la, Lon: lo}
}

func (s *Server) prices() charging.Prices {
	if s.Cfg == nil {
		return charging.Prices{}
	}
	p := charging.Prices{Home: s.Cfg.PriceHome(), Public: s.Cfg.PricePublic()}
	if price, from, to := s.Cfg.OffPeak(); price != nil {
		p.OffPeak, p.Window, p.Loc = price, &charging.Window{From: from, To: to}, s.Cfg.Location()
	}
	return p
}

func (s *Server) currency() string {
	if s.Cfg == nil {
		return "$"
	}
	return s.Cfg.Currency()
}

// effInfo is the slow-charging efficiency in use: measured from sessions where the charger's own number was
// entered (once there are at least three), otherwise the setting.
type effInfo struct {
	Value   float64 `json:"value"`
	Source  string  `json:"source"`  // measured or setting
	Samples int     `json:"samples"` // sessions with the charger's number entered
}

func (s *Server) chargeSessions(limit int) ([]charging.Session, error) {
	out, _, err := s.chargeSessionsEff(limit)
	return out, err
}

func (s *Server) chargeSessionsEff(limit int) ([]charging.Session, effInfo, error) {
	cs, err := s.S.ListCharges(limit)
	if err != nil {
		return nil, effInfo{}, err
	}
	capacity, home, prices := s.capacityKwh(), s.homePoint(), s.prices()
	prices.AcEfficiency = 0.9
	if s.Cfg != nil {
		prices.AcEfficiency = s.Cfg.ChargeEfficiency()
	}
	enrich := func() []charging.Session {
		out := make([]charging.Session, 0, len(cs))
		for _, c := range cs {
			out = append(out, charging.Enrich(c, capacity, home, prices))
		}
		return out
	}
	out := enrich()
	info := effInfo{Value: prices.AcEfficiency, Source: "setting"}
	if eff, n := charging.Learned(out); n > 0 {
		info.Samples = n
		if n >= 3 {
			prices.AcEfficiency = eff
			info.Value, info.Source = eff, "measured"
			out = enrich()
		}
	}
	return out, info, nil
}

// listCharges is every recent charging session with the energy it added and what it cost.
func (s *Server) listCharges(w http.ResponseWriter, r *http.Request) {
	ss, err := s.chargeSessions(200)
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, s.withPlaces(ss))
}

type monthOut struct {
	charging.Month
	Km        float64  `json:"km"`        // driven that month, from the odometer
	CostPerKm *float64 `json:"costPerKm"` // only when every counted session has a cost
}

// chargeSummary totals the last twelve months: sessions, energy, cost, and cost per distance driven.
func (s *Server) chargeSummary(w http.ResponseWriter, r *http.Request) {
	ss, eff, err := s.chargeSessionsEff(1000)
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	loc := s.defaultLoc()
	months := charging.ByMonth(ss, func(ts int64) string { return time.Unix(ts, 0).In(loc).Format("2006-01") })
	if len(months) > 12 {
		months = months[:12]
	}
	out := make([]monthOut, 0, len(months))
	for _, m := range months {
		mo := monthOut{Month: m}
		if first, err := time.ParseInLocation("2006-01", m.Month, loc); err == nil {
			if pts, err := s.S.OdoBetween(first.Unix(), first.AddDate(0, 1, 0).Unix()); err == nil && len(pts) > 1 {
				mo.Km = pts[len(pts)-1].Km - pts[0].Km
			}
		}
		if m.Priced == m.Sessions && m.Priced > 0 && mo.Km > 0 {
			c := m.Cost / mo.Km
			mo.CostPerKm = &c
		}
		out = append(out, mo)
	}
	p := s.prices()
	ok(w, map[string]any{"months": out, "thisMonth": s.monthSoFar(out, loc, time.Now()), "currency": s.currency(), "priceSet": p.Home != nil || p.Public != nil, "homeKnown": s.homePoint() != nil, "efficiency": eff})
}

// patchCharge sets whether a session was at home or public. Marking one as home also tells CarTrak where
// home is, so later sessions there are recognised on their own. {"kind":"home"|"public"|""}. Admin only.
func (s *Server) patchCharge(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	var in struct {
		Kind         *string  `json:"kind"`
		ChargerKwh   *float64 `json:"chargerKwh"`
		ClearCharger bool     `json:"clearCharger"`
		Cost         *float64 `json:"cost"` // what was paid, from the receipt
		ClearCost    bool     `json:"clearCost"`
	}
	if err := decode(r, &in); err != nil || (in.Kind != nil && *in.Kind != "" && *in.Kind != "home" && *in.Kind != "public") {
		fail(w, 400, "choose home, public or automatic")
		return
	}
	c, err := s.S.GetCharge(id)
	if err != nil || c == nil {
		fail(w, 404, "no such charging session")
		return
	}
	if in.Kind != nil {
		if err := s.S.SetChargeKind(id, *in.Kind); err != nil {
			fail(w, 500, "db error")
			return
		}
		if *in.Kind == "home" && c.Lat != nil && c.Lon != nil {
			_ = s.S.SetKV(kvHomeLat, strconv.FormatFloat(*c.Lat, 'f', 5, 64))
			_ = s.S.SetKV(kvHomeLon, strconv.FormatFloat(*c.Lon, 'f', 5, 64))
		}
	}
	if in.ClearCost {
		_ = s.S.SetChargeCost(id, nil)
	} else if in.Cost != nil {
		if *in.Cost < 0 || *in.Cost > 2000 {
			fail(w, 400, "enter what you paid, for example 13.06")
			return
		}
		if err := s.S.SetChargeCost(id, in.Cost); err != nil {
			fail(w, 500, "db error")
			return
		}
	}
	if in.ClearCharger {
		_ = s.S.SetChargerKwh(id, nil)
	} else if in.ChargerKwh != nil {
		added := charging.Enrich(c, s.capacityKwh(), nil, charging.Prices{}).AddedKwh
		switch {
		case *in.ChargerKwh <= 0 || *in.ChargerKwh > 300:
			fail(w, 400, "enter the kWh the charger reported, for example 31.4")
			return
		case added != nil && *in.ChargerKwh < *added*0.95:
			fail(w, 400, "that is less than the energy the battery received, so check the number")
			return
		}
		if err := s.S.SetChargerKwh(id, in.ChargerKwh); err != nil {
			fail(w, 500, "db error")
			return
		}
	}
	ok(w, "saved")
}
