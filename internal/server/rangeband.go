package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
	"github.com/ZachCurry13/cartrak/internal/trips"
)

// intQuery reads a whole-number query value, clamped to a range, with a default.
func intQuery(r *http.Request, name string, def, lo, hi int) int {
	n, err := strconv.Atoi(r.URL.Query().Get(name))
	if err != nil {
		return def
	}
	return min(hi, max(lo, n))
}

type tempBand struct {
	Name     string  `json:"name"`
	Trips    int     `json:"trips"`
	KmPerPct float64 `json:"kmPerPct"`
	BandKm   float64 `json:"bandKm"`
}

type bandMonth struct {
	Month    string  `json:"month"`
	Trips    int     `json:"trips"`
	KmPerPct float64 `json:"kmPerPct"`
	BandKm   float64 `json:"bandKm"`
}

// rangeBand answers "how far do I get between 20 and 80 percent?" from how the car has really been
// driven lately: kilometres per percent of battery over the last `days`, and the same by month.
// GET /api/range-band?low=20&high=80&days=30
func (s *Server) rangeBand(w http.ResponseWriter, r *http.Request) {
	low := intQuery(r, "low", 20, 1, 98)
	high := intQuery(r, "high", 80, low+1, 100)
	days := intQuery(r, "days", 30, 7, 365)
	now := time.Now()
	year, err := s.S.BandTrips(now.AddDate(0, 0, -370).Unix())
	if err != nil {
		fail(w, http.StatusInternalServerError, "could not read trips")
		return
	}
	cutoff := now.AddDate(0, 0, -days).Unix()
	var recent []*store.Trip
	for _, t := range year {
		if t.StartTS >= cutoff {
			recent = append(recent, t)
		}
	}
	out := map[string]any{"low": low, "high": high, "days": days}
	lo, hi := float64(low), float64(high)
	win := trips.Measure(recent)
	if per, ok := win.KmPerPct(); ok {
		out["source"], out["kmPerPct"], out["bandKm"] = "trips", per, trips.KmBetween(per, lo, hi)
		out["trips"], out["km"] = win.Trips, win.Km
	}
	// What the car's own range figure implies, for comparison and for when there is too little driving.
	if x, _ := s.S.LatestSample(); x != nil && x.RangeKm != nil && x.SoCPct != nil && *x.SoCPct >= 10 {
		car := trips.KmBetween(*x.RangeKm / *x.SoCPct, lo, hi)
		out["carBandKm"] = car
		if _, has := out["bandKm"]; !has {
			out["source"], out["bandKm"] = "car", car
		}
	}
	// The same figure by weather: what a stretch of battery goes in cold, mild and warm weather, and for the
	// temperature outside right now.
	byTemp := []tempBand{}
	if temps, err := s.S.TripTemps(now.AddDate(0, 0, -370).Unix()); err == nil {
		for _, b := range trips.MeasureByBand(year, temps) {
			if per, ok := b.KmPerPct(); ok {
				byTemp = append(byTemp, tempBand{Name: b.Band.Name, Trips: b.Trips, KmPerPct: per, BandKm: trips.KmBetween(per, lo, hi)})
			}
		}
	}
	out["byTemp"] = byTemp
	if x, _ := s.S.LatestSample(); x != nil && x.OutsideC != nil {
		out["outsideC"] = *x.OutsideC
		if nb := trips.BandFor(*x.OutsideC); nb != nil {
			for i := range byTemp {
				if byTemp[i].Name == nb.Name {
					out["nowBand"] = byTemp[i]
				}
			}
		}
	}
	months := []bandMonth{}
	for _, m := range trips.ByMonth(year) {
		if per, ok := m.KmPerPct(); ok && len(months) < 12 {
			months = append(months, bandMonth{Month: m.Month, Trips: m.Trips, KmPerPct: per, BandKm: trips.KmBetween(per, lo, hi)})
		}
	}
	out["months"] = months
	ok(w, out)
}
