// Package report turns trips into a spreadsheet: a chosen set of columns, a summary by category, and a second sheet of totals,
// as an Excel workbook or a CSV. The columns and their order can be changed, renamed, and extended with fixed ones, which is
// how custom reports (for an employer, an accountant) are made.
package report

import (
	"encoding/csv"
	"fmt"
	"io"
	"sort"
	"strconv"
	"time"

	"github.com/ZachCurry13/cartrak/internal/xlsx"
)

const kmPerMile = 1.609344

// Column is something a report can show for each trip.
type Column struct {
	Key     string  `json:"key"`
	Label   string  `json:"label"`   // the default heading
	Default bool    `json:"default"` // in the standard report
	Width   float64 `json:"-"`
	Number  bool    `json:"-"`
}

// Columns lists every column, in the order the standard report shows them.
var Columns = []Column{
	{Key: "value", Label: "Value", Default: true, Width: 12, Number: true},
	{Key: "distance", Label: "Miles", Default: true, Width: 10, Number: true},
	{Key: "rate", Label: "Mileage Rate", Default: true, Width: 13, Number: true},
	{Key: "from", Label: "From", Default: true, Width: 28},
	{Key: "to", Label: "To", Default: true, Width: 28},
	{Key: "date", Label: "Date", Default: true, Width: 12},
	{Key: "purpose", Label: "Purpose", Default: true, Width: 12},
	{Key: "track", Label: "Track Method", Default: true, Width: 14},
	{Key: "notes", Label: "Notes", Default: true, Width: 24},
	{Key: "vehicle", Label: "Vehicle", Default: true, Width: 16},
	{Key: "started", Label: "Time Started", Default: true, Width: 13},
	{Key: "ended", Label: "Time Ended", Default: true, Width: 13},
	{Key: "hours", Label: "Duration (hours)", Default: true, Width: 16, Number: true},
	{Key: "zone", Label: "Time Zone", Default: true, Width: 22},
	{Key: "weekday", Label: "Day of Week", Width: 12},
	{Key: "energy", Label: "Energy Used (kWh)", Width: 17, Number: true},
	{Key: "efficiency", Label: "Efficiency", Width: 14},
	{Key: "energycost", Label: "Energy Cost", Width: 13, Number: true},
	{Key: "costper", Label: "Cost per Mile", Width: 14, Number: true},
	{Key: "mpge", Label: "MPGe", Width: 9, Number: true},
	{Key: "avgspeed", Label: "Average Speed", Width: 14, Number: true},
	{Key: "socstart", Label: "Battery Start (%)", Width: 16, Number: true},
	{Key: "socend", Label: "Battery End (%)", Width: 16, Number: true},
	{Key: "odostart", Label: "Odometer Start", Width: 15, Number: true},
	{Key: "odoend", Label: "Odometer End", Width: 15, Number: true},
	{Key: "topspeed", Label: "Top Speed", Width: 11, Number: true},
	{Key: "id", Label: "Trip ID", Width: 8, Number: true},
}

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

var byKey = func() map[string]Column {
	m := map[string]Column{}
	for _, c := range Columns {
		m[c.Key] = c
	}
	return m
}()

// ValidKey says whether a column key exists.
func ValidKey(k string) bool { _, ok := byKey[k]; return ok }

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

// cell is what one column shows for one trip.
func (o Options) cell(key string, t Trip) xlsx.Cell {
	n := func(v float64, d int) xlsx.Cell { return xlsx.N(round(v, d)) }
	np := func(p *float64, d int) xlsx.Cell {
		if p == nil {
			return xlsx.S("")
		}
		return n(*p, d)
	}
	switch key {
	case "value":
		return n(o.value(t), 2)
	case "distance":
		return n(o.dist(t.DistanceKm), 1)
	case "rate":
		return n(o.rate(t.Category), 3)
	case "from":
		return xlsx.S(t.From)
	case "to":
		return xlsx.S(t.To)
	case "date":
		return xlsx.S(t.Start.Format("2006-01-02"))
	case "purpose":
		if l := o.Labels[t.Category]; l != "" {
			return xlsx.S(l)
		}
		return xlsx.S(t.Category)
	case "track":
		return xlsx.S(t.Method)
	case "notes":
		return xlsx.S(t.Notes)
	case "vehicle":
		return xlsx.S(o.Vehicle)
	case "started":
		return xlsx.S(t.Start.Format("03:04 PM"))
	case "ended":
		return xlsx.S(t.End.Format("03:04 PM"))
	case "hours":
		return n(t.End.Sub(t.Start).Hours(), 1)
	case "zone":
		return xlsx.S(t.Zone)
	case "weekday":
		return xlsx.S(t.Start.Weekday().String())
	case "energy":
		return np(t.EnergyKwh, 2)
	case "efficiency":
		if t.EnergyKwh == nil || *t.EnergyKwh <= 0 || t.DistanceKm < 0.5 {
			return xlsx.S("")
		}
		return xlsx.S(o.effText(t.DistanceKm, *t.EnergyKwh))
	case "energycost":
		if t.EnergyKwh == nil || o.KwhPrice <= 0 {
			return xlsx.S("")
		}
		return n(*t.EnergyKwh*o.KwhPrice, 2)
	case "costper":
		if t.EnergyKwh == nil || o.KwhPrice <= 0 || t.DistanceKm < 0.5 {
			return xlsx.S("")
		}
		return n(*t.EnergyKwh*o.KwhPrice/o.dist(t.DistanceKm), 3)
	case "mpge":
		if t.EnergyKwh == nil || *t.EnergyKwh <= 0 || t.DistanceKm < 0.5 {
			return xlsx.S("")
		}
		return n(t.DistanceKm/kmPerMile/(*t.EnergyKwh/33.705), 0) // 33.705 kWh is the energy of a gallon of gasoline
	case "avgspeed":
		if h := t.End.Sub(t.Start).Hours(); h > 0 {
			return n(o.dist(t.DistanceKm)/h, 0)
		}
		return xlsx.S("")
	case "socstart":
		return np(t.SoCStart, 0)
	case "socend":
		return np(t.SoCEnd, 0)
	case "odostart":
		if t.OdoStartKm == nil {
			return xlsx.S("")
		}
		return n(o.dist(*t.OdoStartKm), 1)
	case "odoend":
		if t.OdoEndKm == nil {
			return xlsx.S("")
		}
		return n(o.dist(*t.OdoEndKm), 1)
	case "topspeed":
		return n(o.dist(t.TopSpeedKph), 0)
	case "id":
		return xlsx.N(float64(t.ID))
	}
	return xlsx.S("")
}

// effText writes how far a trip went on its energy, in the unit chosen: miles per kWh, watt-hours per mile, or miles per percent of
// the battery (kilometres when metric; kWh per 100 km stands in for the first, as the app does).
func (o Options) effText(km, kwh float64) string {
	switch {
	case o.EffUnit == "pct" && o.Capacity > 0:
		used := kwh / (o.Capacity / 100)
		if o.Metric {
			return fmt.Sprintf("%.2f km/%%", km/used)
		}
		return fmt.Sprintf("%.2f mi/%%", km/kmPerMile/used)
	case o.EffUnit == "wh":
		if o.Metric {
			return fmt.Sprintf("%d Wh/km", int(kwh*1000/km+0.5))
		}
		return fmt.Sprintf("%d Wh/mi", int(kwh*1000/(km/kmPerMile)+0.5))
	case o.Metric:
		return fmt.Sprintf("%.1f kWh/100 km", kwh/km*100)
	}
	return fmt.Sprintf("%.2f mi/kWh", km/kmPerMile/kwh)
}

// Totals is what a category added up to.
type Totals struct {
	Key      string
	Label    string
	Trips    int
	Distance float64 // in the report's unit
	Value    float64
}

// Sum adds the trips up by category, in the order of the labels given (then any others), with a grand total last.
func Sum(trips []Trip, o Options) (rows []Totals, total Totals) {
	by := map[string]*Totals{}
	for _, t := range trips {
		x := by[t.Category]
		if x == nil {
			l := o.Labels[t.Category]
			if l == "" {
				l = t.Category
			}
			x = &Totals{Key: t.Category, Label: l}
			by[t.Category] = x
		}
		x.Trips++
		x.Distance += o.dist(t.DistanceKm)
		x.Value += o.value(t)
	}
	for _, x := range by {
		rows = append(rows, *x)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Label < rows[j].Label })
	total.Label = "TOTAL"
	for _, x := range rows {
		total.Trips += x.Trips
		total.Distance += x.Distance
		total.Value += x.Value
	}
	return rows, total
}

// header and rows build the table: the heading row and one row per trip, oldest first.
func (o Options) table(trips []Trip) [][]xlsx.Cell {
	picks := o.picks()
	head := make([]xlsx.Cell, 0, len(picks)+len(o.Fixed))
	for _, p := range picks {
		head = append(head, xlsx.B(p.Label))
	}
	for _, f := range o.Fixed {
		head = append(head, xlsx.B(f.Label))
	}
	out := [][]xlsx.Cell{head}
	ts := append([]Trip(nil), trips...)
	sort.SliceStable(ts, func(i, j int) bool { return ts[i].Start.Before(ts[j].Start) })
	for _, t := range ts {
		row := make([]xlsx.Cell, 0, len(head))
		for _, p := range picks {
			row = append(row, o.cell(p.Key, t))
		}
		for _, f := range o.Fixed {
			row = append(row, xlsx.S(f.Value))
		}
		out = append(out, row)
	}
	return out
}

func (o Options) widths() []float64 {
	var w []float64
	for _, p := range o.picks() {
		w = append(w, byKey[p.Key].Width)
	}
	for range o.Fixed {
		w = append(w, 16)
	}
	return w
}

// Workbook builds the Excel file: a Trips sheet (the summary block, then the table) and a Totals By Label sheet.
func Workbook(trips []Trip, o Options) *xlsx.Workbook {
	unit := "Miles"
	if o.Metric {
		unit = "Kilometres"
	}
	rows, total := Sum(trips, o)
	sheet := &xlsx.Sheet{Name: "Trips", Widths: o.widths()}
	if o.Summary {
		title := o.Title
		if title == "" {
			title = "CarTrak Trip Report"
		}
		sheet.Rows = append(sheet.Rows, []xlsx.Cell{xlsx.B(title)}, nil,
			[]xlsx.Cell{xlsx.S("Name:"), xlsx.S(o.Person)},
			[]xlsx.Cell{xlsx.S("Vehicle:"), xlsx.S(o.Vehicle)},
			[]xlsx.Cell{xlsx.S("Date Range:"), xlsx.S(o.Range)},
			[]xlsx.Cell{xlsx.S("Purpose:"), xlsx.S(o.Purpose)},
			nil, []xlsx.Cell{xlsx.B("Summary")}, []xlsx.Cell{xlsx.S(""), xlsx.B(unit), xlsx.B("Value")})
		for _, r := range rows {
			sheet.Rows = append(sheet.Rows, []xlsx.Cell{xlsx.S(r.Label), xlsx.N(round(r.Distance, 1)), xlsx.N(round(r.Value, 2))})
		}
		sheet.Rows = append(sheet.Rows, []xlsx.Cell{xlsx.B("TOTAL"), xlsx.N(round(total.Distance, 1)), xlsx.N(round(total.Value, 2))}, nil)
	}
	tbl := o.table(trips)
	if len(tbl) > 0 {
		sheet.FilterRow = len(sheet.Rows) + 1
	}
	sheet.Rows = append(sheet.Rows, tbl...)
	totals := &xlsx.Sheet{Name: "Totals By Label", Widths: []float64{24, 10, 14, 12}, FilterRow: 1,
		Rows: [][]xlsx.Cell{{xlsx.B("Category"), xlsx.B("Trips"), xlsx.B(unit), xlsx.B("Value")}}}
	for _, r := range rows {
		totals.Rows = append(totals.Rows, []xlsx.Cell{xlsx.S(r.Label), xlsx.N(float64(r.Trips)), xlsx.N(round(r.Distance, 1)), xlsx.N(round(r.Value, 2))})
	}
	totals.Rows = append(totals.Rows, []xlsx.Cell{xlsx.B("Total"), xlsx.N(float64(total.Trips)), xlsx.N(round(total.Distance, 1)), xlsx.N(round(total.Value, 2))})
	return &xlsx.Workbook{Sheets: []*xlsx.Sheet{sheet, totals}}
}

// CSV writes only the table: the heading row and one row per trip.
func CSV(w io.Writer, trips []Trip, o Options) error {
	cw := csv.NewWriter(w)
	for _, row := range o.table(trips) {
		rec := make([]string, len(row))
		for i, c := range row {
			switch {
			case c.Num != nil:
				rec[i] = strconv.FormatFloat(*c.Num, 'f', -1, 64)
			default:
				rec[i] = safe(c.Text)
			}
		}
		if err := cw.Write(rec); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// safe stops a spreadsheet from running text that begins like a formula.
func safe(s string) string {
	if s != "" && (s[0] == '=' || s[0] == '+' || s[0] == '-' || s[0] == '@') {
		return "'" + s
	}
	return s
}
