package server

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ZachCurry13/cartrak/internal/charging"
	"github.com/ZachCurry13/cartrak/internal/history"
)

const kvHomeLat, kvHomeLon = "home_lat", "home_lon"

// homePoint is where home is: the place the owner named Home (More, Places), or else where they marked a session as home. Nil
// until one of those exists.
func (s *Server) homePoint() *charging.Point {
	if p := s.namedHome(); p != nil {
		return p
	}
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
	if ps, err := s.S.ListNamedPlaces(); err == nil {
		for _, x := range ps {
			if x.ChargePrice != nil || x.ChargeHome || strings.EqualFold(strings.TrimSpace(x.Name), "home") {
				p.Places = append(p.Places, charging.PlaceRate{Name: x.Name, Lat: x.Lat, Lon: x.Lon, RadiusM: x.RadiusM, Price: x.ChargePrice, Home: x.ChargeHome || strings.EqualFold(strings.TrimSpace(x.Name), "home")})
			}
		}
	}
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
	out := s.withPlaces(ss)
	eq := s.gasEquivalents(ss)
	for i := range out {
		if g, ok := eq[out[i].ID]; ok {
			out[i].Gas = &history.GasCompare{Cost: g.GasCost, Km: g.Km, Source: g.Source}
		}
	}
	ok(w, out)
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
