package server

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/ZachCurry13/cartrak/internal/fuel"
)

// The fuel page of a gas or hybrid car: its fill-ups, the tank size learned from them, and what each month cost. Amounts are kept
// in US gallons and the price per gallon; the page converts for people who use liters.

type fillOut struct {
	fuel.Fill
	Cost     *float64 `json:"cost"`
	Km       float64  `json:"km"`       // driven since the fill-up before, by the odometer
	KmPerGal float64  `json:"kmPerGal"` // how far a gallon went since the fill-up before; 0 when it cannot be told
	// The live average price of a gallon where and when it was bought, when live gas prices are on.
	AreaPerGal *float64 `json:"areaPerGal"`
	AreaSource string   `json:"areaSource,omitempty"`
}

type fuelMonth struct {
	Month   string  `json:"month"`
	Gallons float64 `json:"gallons"`
	Cost    float64 `json:"cost"`
	Fills   int     `json:"fills"`
}

// fuelPage: GET /api/fuel
func (s *Server) fuelPage(w http.ResponseWriter, r *http.Request) {
	fills, err := s.S.ListRefuels()
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	tank, n := fuel.TankGallons(fills)
	loc := s.defaultLoc()
	out := make([]fillOut, 0, len(fills))
	months := map[string]*fuelMonth{}
	for i, f := range fills {
		o := fillOut{Fill: f, Cost: f.Cost()}
		if f.PricePerGal != nil {
			lat, lon := s.S.LastPositionBefore(f.TS)
			if q := s.gasQuote(lat, lon, f.TS); q.OK {
				o.AreaPerGal, o.AreaSource = &q.PerGallon, q.Source
			}
		}
		if i > 0 {
			o.Km, _, o.KmPerGal = fuel.Between(fills[i-1], f, tank)
		}
		out = append(out, o)
		if f.Gallons != nil {
			key := time.Unix(f.TS, 0).In(loc).Format("2006-01")
			m := months[key]
			if m == nil {
				m = &fuelMonth{Month: key}
				months[key] = m
			}
			m.Gallons += *f.Gallons
			m.Fills++
			if c := f.Cost(); c != nil {
				m.Cost += *c
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TS > out[j].TS }) // newest first
	list := make([]fuelMonth, 0, len(months))
	for _, m := range months {
		list = append(list, *m)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Month > list[j].Month })
	var tankPtr *float64
	if n > 0 {
		tankPtr = &tank
	}
	ok(w, map[string]any{"tankGal": tankPtr, "tankFills": n, "fills": out, "months": list, "pending": s.S.PendingRefuels(),
		"recentPrice": recentFuelPrice(fills), "currency": s.currency()})
}

type fillIn struct {
	TS          int64    `json:"ts"`
	Gallons     *float64 `json:"gallons"`
	PricePerGal *float64 `json:"pricePerGal"`
	StartPct    *float64 `json:"startPct"`
	EndPct      *float64 `json:"endPct"`
	OdoKm       *float64 `json:"odoKm"`
	Status      string   `json:"status"`
}

func okAmount(v *float64, most float64) bool { return v == nil || (*v > 0 && *v <= most) }
func okPct(v *float64) bool                  { return v == nil || (*v >= 0 && *v <= 100) }

// addFill: POST /api/fuel/fills, a fill-up entered by hand.
func (s *Server) addFill(w http.ResponseWriter, r *http.Request) {
	var in fillIn
	if decode(r, &in) != nil || in.Gallons == nil || !okAmount(in.Gallons, 80) || !okAmount(in.PricePerGal, 50) || !okPct(in.StartPct) || !okPct(in.EndPct) {
		fail(w, 400, "enter the gallons (up to 80) and the price per gallon")
		return
	}
	f := &fuel.Fill{TS: in.TS, Gallons: in.Gallons, PricePerGal: in.PricePerGal, OdoKm: in.OdoKm, Status: "logged", Source: "manual"}
	if f.TS <= 0 || f.TS > time.Now().Unix()+3600 {
		f.TS = time.Now().Unix()
	}
	if in.StartPct != nil && in.EndPct != nil && *in.EndPct > *in.StartPct {
		f.StartPct, f.EndPct = *in.StartPct, *in.EndPct
	}
	if f.OdoKm == nil {
		if x, _ := s.S.LatestSample(); x != nil {
			f.OdoKm = x.OdometerKm
		}
	}
	if err := s.S.SaveRefuel(f); err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, f)
}

// patchFill: PATCH /api/fuel/fills/{id}. Completes a found fill-up with the gallons and the price, or dismisses it.
func (s *Server) patchFill(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	f, err := s.S.GetRefuel(id)
	var in fillIn
	if err != nil || f == nil || decode(r, &in) != nil || !okAmount(in.Gallons, 80) || !okAmount(in.PricePerGal, 50) || !okPct(in.StartPct) || !okPct(in.EndPct) {
		fail(w, 400, "no such fill-up, or an amount that does not look right")
		return
	}
	if in.Gallons != nil {
		f.Gallons = in.Gallons
	}
	if in.PricePerGal != nil {
		f.PricePerGal = in.PricePerGal
	}
	if in.StartPct != nil && in.EndPct != nil && *in.EndPct > *in.StartPct {
		f.StartPct, f.EndPct = *in.StartPct, *in.EndPct
	}
	switch {
	case in.Status == "dismissed":
		f.Status = "dismissed"
	case f.Gallons != nil:
		f.Status = "logged"
	}
	if err := s.S.SaveRefuel(f); err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, f)
}

// deleteFill: DELETE /api/fuel/fills/{id}
func (s *Server) deleteFill(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err := s.S.DeleteRefuel(id); err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, "deleted")
}

// fuelDebug is the fuel part of the debug report: nothing for a car that has no fuel gauge readings.
func (s *Server) fuelDebug(from time.Time) []string {
	pts, err := s.S.FuelBetween(from.Unix(), time.Now().Unix())
	fills, _ := s.S.ListRefuels()
	if err != nil || (len(pts) == 0 && len(fills) == 0) {
		return nil
	}
	loc := s.defaultLoc()
	at := func(ts int64) string { return time.Unix(ts, 0).In(loc).Format("01-02 15:04") }
	var out []string
	if tank, n := fuel.TankGallons(fills); n > 0 {
		out = append(out, fmt.Sprintf("tank size %.1f gal, from %d fill-up(s)", tank, n))
	} else {
		out = append(out, "tank size not known yet (no fill-up with its gallons)")
	}
	out = append(out, fmt.Sprintf("fuel readings in this window: %d", len(pts)))
	for _, f := range fills {
		gal, price := "-", "-"
		if f.Gallons != nil {
			gal = fmt.Sprintf("%.2f gal", *f.Gallons)
		}
		if f.PricePerGal != nil {
			price = fmt.Sprintf("%.3f/gal", *f.PricePerGal)
		}
		out = append(out, fmt.Sprintf("fill-up %s | tank %.0f%% > %.0f%% | %s | %s | %s, %s", at(f.TS), f.StartPct, f.EndPct, gal, price, f.Status, f.Source))
	}
	step := 1
	if len(pts) > 60 { // a line every so often keeps the report short
		step = len(pts) / 60
	}
	for i := 0; i < len(pts); i += step {
		out = append(out, fmt.Sprintf("gauge %s | %.0f%%", at(pts[i].TS), pts[i].Pct))
	}
	return out
}
