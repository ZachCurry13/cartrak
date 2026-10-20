package report

import (
	"encoding/csv"
	"io"
	"strconv"

	"github.com/ZachCurry13/cartrak/internal/mileage"
	"github.com/ZachCurry13/cartrak/internal/xlsx"
)

var mileageHeads = []string{"Date", "Origin City", "Destination City", "Description", "Total Miles Driven"}

// MileageWorkbook is the mileage log as an Excel file: the heading row, one row a day, and the total at the end.
func MileageWorkbook(rep mileage.Report) *xlsx.Workbook {
	sh := &xlsx.Sheet{Name: "Mileage Log", Widths: []float64{12, 24, 48, 22, 20}, FilterRow: 1}
	head := make([]xlsx.Cell, len(mileageHeads))
	for i, h := range mileageHeads {
		head[i] = xlsx.B(h)
	}
	sh.Rows = append(sh.Rows, head)
	for _, r := range rep.Rows {
		sh.Rows = append(sh.Rows, []xlsx.Cell{xlsx.S(r.Date), xlsx.S(r.Origin), xlsx.S(r.Destinations), xlsx.S(r.Description), xlsx.N(float64(r.Miles))})
	}
	sh.Rows = append(sh.Rows, nil, []xlsx.Cell{xlsx.B("Total Mileage Recorded:"), xlsx.S(""), xlsx.S(""), xlsx.S(""), xlsx.N(float64(rep.TotalMiles))})
	for _, n := range rep.Notes {
		sh.Rows = append(sh.Rows, nil, []xlsx.Cell{xlsx.S(n)})
	}
	return &xlsx.Workbook{Sheets: []*xlsx.Sheet{sh}}
}

// MileageCSV writes the same log as CSV.
func MileageCSV(w io.Writer, rep mileage.Report) error {
	cw := csv.NewWriter(w)
	_ = cw.Write(mileageHeads)
	for _, r := range rep.Rows {
		_ = cw.Write([]string{safe(r.Date), safe(r.Origin), safe(r.Destinations), safe(r.Description), strconv.Itoa(r.Miles)})
	}
	_ = cw.Write(nil)
	_ = cw.Write([]string{"Total Mileage Recorded:", "", "", "", strconv.Itoa(rep.TotalMiles)})
	cw.Flush()
	return cw.Error()
}
