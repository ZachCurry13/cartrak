// Package report turns trips into a spreadsheet: a chosen set of columns, a summary by category, and a second sheet of totals,
// as an Excel workbook or a CSV. The columns and their order can be changed, renamed, and extended with fixed ones, which is
// how custom reports (for an employer, an accountant) are made.
package report

import (
	"encoding/csv"
	"io"
	"sort"
	"strconv"

	"github.com/ZachCurry13/cartrak/internal/xlsx"
)

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
