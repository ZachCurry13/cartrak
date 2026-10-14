package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/ZachCurry13/cartrak/internal/yearreport"
)

// chargeYear is the charging part of the yearly summary.
type chargeYear struct {
	Sessions    int     `json:"sessions"`
	AddedKwh    float64 `json:"addedKwh"` // into the battery
	WallKwh     float64 `json:"wallKwh"`  // from the charger, with losses
	Cost        float64 `json:"cost"`
	Priced      int     `json:"priced"`
	HomeKwh     float64 `json:"homeKwh"`
	PublicKwh   float64 `json:"publicKwh"`
	LossPercent *int    `json:"lossPercent"`
}

// yearSummary is a year of driving and charging in one place. GET /api/report/year?year=2026 (default: this year).
func (s *Server) yearSummary(w http.ResponseWriter, r *http.Request) {
	loc := s.defaultLoc()
	now := time.Now().In(loc)
	year := now.Year()
	if v := r.URL.Query().Get("year"); v != "" {
		y, err := strconv.Atoi(v)
		if err != nil || y < 2000 || y > now.Year() {
			fail(w, 400, "bad year")
			return
		}
		year = y
	}
	from := time.Date(year, 1, 1, 0, 0, 0, 0, loc)
	to := from.AddDate(1, 0, 0)
	ts, err := s.S.YearTrips(from.Unix(), to.Unix())
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	drive := yearreport.Build(ts, s.capacityKwh(), loc)
	var odoKm float64
	if pts, err := s.S.OdoBetween(from.Unix(), to.Unix()); err == nil && len(pts) > 1 {
		odoKm = pts[len(pts)-1].Km - pts[0].Km
	}
	var ch chargeYear
	if ss, _, err := s.chargeSessionsEff(1000); err == nil {
		for _, c := range ss {
			if c.Open || c.AddedKwh == nil || c.StartTS < from.Unix() || c.StartTS >= to.Unix() {
				continue
			}
			ch.Sessions++
			ch.AddedKwh += *c.AddedKwh
			if c.WallKwh != nil {
				ch.WallKwh += *c.WallKwh
			}
			if c.Cost != nil {
				ch.Cost += *c.Cost
				ch.Priced++
			}
			if c.Where == "home" {
				ch.HomeKwh += *c.AddedKwh
			} else if c.Where == "public" {
				ch.PublicKwh += *c.AddedKwh
			}
		}
		if ch.WallKwh > 0 && ch.AddedKwh > 0 && ch.WallKwh >= ch.AddedKwh {
			p := int((1 - ch.AddedKwh/ch.WallKwh) * 100)
			ch.LossPercent = &p
		}
	}
	first := s.S.FirstTripTS()
	years := []int{}
	if first > 0 {
		for y := time.Unix(first, 0).In(loc).Year(); y <= now.Year(); y++ {
			years = append(years, y)
		}
	}
	if len(years) == 0 {
		years = []int{now.Year()}
	}
	ok(w, map[string]any{"year": year, "years": years, "driving": drive, "odometerKm": odoKm, "charging": ch,
		"currency": s.currency(), "partial": year == now.Year()})
}
