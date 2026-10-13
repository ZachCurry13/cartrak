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
	return charging.Prices{Home: s.Cfg.PriceHome(), Public: s.Cfg.PricePublic()}
}

func (s *Server) currency() string {
	if s.Cfg == nil {
		return "$"
	}
	return s.Cfg.Currency()
}

func (s *Server) chargeSessions(limit int) ([]charging.Session, error) {
	cs, err := s.S.ListCharges(limit)
	if err != nil {
		return nil, err
	}
	capacity, home, prices := s.capacityKwh(), s.homePoint(), s.prices()
	out := make([]charging.Session, 0, len(cs))
	for _, c := range cs {
		out = append(out, charging.Enrich(c, capacity, home, prices))
	}
	return out, nil
}

// listCharges is every recent charging session with the energy it added and what it cost.
func (s *Server) listCharges(w http.ResponseWriter, r *http.Request) {
	ss, err := s.chargeSessions(200)
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, ss)
}

type monthOut struct {
	charging.Month
	Km        float64  `json:"km"`        // driven that month, from the odometer
	CostPerKm *float64 `json:"costPerKm"` // only when every counted session has a cost
}

// chargeSummary totals the last twelve months: sessions, energy, cost, and cost per distance driven.
func (s *Server) chargeSummary(w http.ResponseWriter, r *http.Request) {
	ss, err := s.chargeSessions(1000)
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
	ok(w, map[string]any{"months": out, "currency": s.currency(), "priceSet": p.Home != nil || p.Public != nil, "homeKnown": s.homePoint() != nil})
}

// patchCharge sets whether a session was at home or public. Marking one as home also tells CarTrak where
// home is, so later sessions there are recognised on their own. {"kind":"home"|"public"|""}. Admin only.
func (s *Server) patchCharge(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	var in struct {
		Kind string `json:"kind"`
	}
	if err := decode(r, &in); err != nil || (in.Kind != "" && in.Kind != "home" && in.Kind != "public") {
		fail(w, 400, "choose home, public or automatic")
		return
	}
	c, err := s.S.GetCharge(id)
	if err != nil || c == nil {
		fail(w, 404, "no such charging session")
		return
	}
	if err := s.S.SetChargeKind(id, in.Kind); err != nil {
		fail(w, 500, "db error")
		return
	}
	if in.Kind == "home" && c.Lat != nil && c.Lon != nil {
		_ = s.S.SetKV(kvHomeLat, strconv.FormatFloat(*c.Lat, 'f', 5, 64))
		_ = s.S.SetKV(kvHomeLon, strconv.FormatFloat(*c.Lon, 'f', 5, 64))
	}
	ok(w, map[string]any{"kind": in.Kind})
}
