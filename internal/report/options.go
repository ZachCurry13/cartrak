package report

import (
	"time"
)

// Trip is one trip, flattened for a report.
type Trip struct {
	ID          int64
	Start, End  time.Time // in the trip's own time zone
	Zone        string
	Category    string
	DistanceKm  float64
	Notes       string
	From, To    string
	Method      string // "Car" or "Phone GPS"
	EnergyKwh   *float64
	SoCStart    *float64
	SoCEnd      *float64
	OdoStartKm  *float64
	OdoEndKm    *float64
	TopSpeedKph float64
}

// Pick is a column in a report, with the heading to show for it.
type Pick struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

// Fixed is a column whose value is the same on every row (a company name, a cost centre).
type Fixed struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// Options says what the report looks like and how it is worked out.
type Options struct {
	Metric   bool               // kilometres and km/h; otherwise miles and mph
	Columns  []Pick             // the columns, in order; empty means the standard ones
	Fixed    []Fixed            // extra columns with a fixed value
	Labels   map[string]string  // category key to its name
	Rates    map[string]float64 // category key to a deduction per mile
	Vehicle  string
	Person   string
	Range    string // for the heading, for example "January 1, 2026 - October 20, 2026"
	Purpose  string // the categories chosen, for the heading
	Summary  bool   // the summary block above the table, and the totals sheet
	Title    string // the first line of the summary block
	Currency string
	EffUnit  string  // how efficiency is shown: "kwh" (mi per kWh, or kWh per 100 km), "wh" (Wh per mile or km) or "pct" (distance per percent of battery)
	Capacity float64 // the battery size in kWh, for "pct"
	KwhPrice float64 // what a kWh of the energy a trip used cost, from the charging in the period; 0 when not known
}

func (o Options) picks() []Pick {
	if len(o.Columns) > 0 {
		var out []Pick
		for _, p := range o.Columns {
			if c, ok := byKey[p.Key]; ok {
				if p.Label == "" {
					p.Label = c.Label
					if p.Key == "distance" && o.Metric {
						p.Label = "Kilometres"
					}
					p.Label = o.label(c)
				}
				out = append(out, p)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	var out []Pick
	for _, c := range Columns {
		if c.Default {
			out = append(out, Pick{Key: c.Key, Label: o.label(c)})
		}
	}
	return out
}

func (o Options) label(c Column) string {
	switch {
	case c.Key == "distance" && o.Metric:
		return "Kilometres"
	case c.Key == "topspeed":
		return "Top Speed (" + o.speedUnit() + ")"
	case c.Key == "costper" && o.Metric:
		return "Cost per Kilometre"
	case c.Key == "avgspeed":
		return "Average Speed (" + o.speedUnit() + ")"
	}
	return c.Label
}

func (o Options) speedUnit() string {
	if o.Metric {
		return "km/h"
	}
	return "mph"
}

func (o Options) dist(km float64) float64 {
	if o.Metric {
		return km
	}
	return km / kmPerMile
}

func round(v float64, d int) float64 {
	p := 1.0
	for i := 0; i < d; i++ {
		p *= 10
	}
	return float64(int64(v*p+copysignHalf(v))) / p
}

func copysignHalf(v float64) float64 {
	if v < 0 {
		return -0.5
	}
	return 0.5
}

// rate is the deduction per mile for a trip's category.
func (o Options) rate(cat string) float64 { return o.Rates[cat] }

func (o Options) value(t Trip) float64 { return t.DistanceKm / kmPerMile * o.rate(t.Category) }
