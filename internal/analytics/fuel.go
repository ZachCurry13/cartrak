package analytics

import (
	"sort"
	"time"

	"github.com/ZachCurry13/cartrak/internal/fuel"
)

// FuelRecap is what a gas or hybrid car burned and spent over a stretch of time. Gallons burned come from the trips (the tank
// gauge over each drive, once the tank size is known); what was spent comes from the fill-ups, where the price per gallon is
// entered each time.
type FuelRecap struct {
	Period    string  `json:"period,omitempty"`
	Gal       float64 `json:"gal"`       // burned on trips with a gauge reading
	KmFueled  float64 `json:"kmFueled"`  // how far those trips went
	KmPerGal  float64 `json:"kmPerGal"`  // 0 when there is nothing to divide
	GalBought float64 `json:"galBought"` // put in at fill-ups
	Spent     float64 `json:"spent"`
	Fills     int     `json:"fills"`
	AvgPrice  float64 `json:"avgPrice"`  // per gallon, weighted by the gallons
	CostPerKm float64 `json:"costPerKm"` // spent over all the driving in the stretch; 0 when nothing was spent
}

// Any says whether there is anything to tell.
func (f FuelRecap) Any() bool { return f.Gal > 0 || f.Fills > 0 }

// FuelSummary totals the trips and fill-ups that started in [from, to).
func FuelSummary(from, to int64, ts []Trip, fills []fuel.Fill) FuelRecap {
	var r FuelRecap
	var km, priced float64
	for _, t := range ts {
		if t.Start < from || t.Start >= to || t.Km < 0.3 {
			continue
		}
		km += t.Km
		if t.Gal > 0 {
			r.Gal += t.Gal
			r.KmFueled += t.Km
		}
	}
	for _, f := range fills {
		if f.TS < from || f.TS >= to || f.Gallons == nil || *f.Gallons <= 0 {
			continue
		}
		r.Fills++
		r.GalBought += *f.Gallons
		if c := f.Cost(); c != nil {
			r.Spent += *c
			priced += *f.Gallons
		}
	}
	if r.Gal > 0 {
		r.KmPerGal = r.KmFueled / r.Gal
	}
	if priced > 0 {
		r.AvgPrice = r.Spent / priced
	}
	if r.Spent > 0 && km > 0 {
		r.CostPerKm = r.Spent / km
	}
	return r
}

// FuelMonths is FuelSummary for each calendar month that has trips with fuel or fill-ups, newest first.
func FuelMonths(ts []Trip, fills []fuel.Fill, loc *time.Location) []FuelRecap {
	seen := map[string]bool{}
	for _, t := range ts {
		if t.Gal > 0 {
			seen[time.Unix(t.Start, 0).In(loc).Format("2006-01")] = true
		}
	}
	for _, f := range fills {
		seen[time.Unix(f.TS, 0).In(loc).Format("2006-01")] = true
	}
	var out []FuelRecap
	for m := range seen {
		from, to, _, ok := Period(m, loc)
		if !ok {
			continue
		}
		r := FuelSummary(from, to, ts, fills)
		r.Period = m
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Period > out[j].Period })
	return out
}

// kmPerGalCeiling is the most a drive may get per gallon and still count as a best: 1.6 times what the car usually gets, so one
// gauge that fell too little over a drive does not become the record (the same care as EffCeiling for electric cars).
func kmPerGalCeiling(ts []Trip) float64 {
	var effs []float64
	for _, t := range ts {
		if t.Gal > 0 && t.Km >= 16 {
			effs = append(effs, t.Km/t.Gal)
		}
	}
	if len(effs) < 5 {
		return 1e9
	}
	sort.Float64s(effs)
	return 1.6 * effs[len(effs)/2]
}

// FuelRecords are the personal bests of a gas or hybrid car: the best economy on a drive of 10 miles or more, the farthest between
// two fill-ups, the biggest fill-up and the cheapest gallon.
func FuelRecords(ts []Trip, fills []fuel.Fill) []Record {
	var out []Record
	ceiling := kmPerGalCeiling(ts)
	var best *Record
	for _, t := range ts {
		if t.Gal < 0.3 || t.Km < 16 || t.Corrected {
			continue
		}
		if e := t.Km / t.Gal; e < ceiling && (best == nil || e > best.Value) {
			best = &Record{Key: "bestMpg", Title: "Best fuel economy (10 mi or more)", Kind: "kmPerGal", Value: e, TS: t.Start, TripID: t.ID}
		}
	}
	if best != nil {
		out = append(out, *best)
	}
	var far, big, cheap *Record
	for i, f := range fills {
		if f.Gallons == nil || *f.Gallons <= 0 {
			continue
		}
		if big == nil || *f.Gallons > big.Value {
			big = &Record{Key: "biggestFill", Title: "Biggest fill-up", Kind: "gal", Value: *f.Gallons, TS: f.TS}
		}
		if f.PricePerGal != nil && (cheap == nil || *f.PricePerGal < cheap.Value) {
			cheap = &Record{Key: "cheapestGas", Title: "Cheapest gas", Kind: "pricePerGal", Value: *f.PricePerGal, TS: f.TS}
		}
		if i > 0 && fills[i-1].OdoKm != nil && f.OdoKm != nil {
			if km := *f.OdoKm - *fills[i-1].OdoKm; km >= 50 && (far == nil || km > far.Value) {
				far = &Record{Key: "longestTank", Title: "Farthest between fill-ups", Kind: "km", Value: km, TS: f.TS}
			}
		}
	}
	for _, r := range []*Record{far, big, cheap} {
		if r != nil {
			out = append(out, *r)
		}
	}
	return out
}
