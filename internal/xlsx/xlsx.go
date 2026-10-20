// Package xlsx writes a small Excel workbook (.xlsx): sheets of text and number cells, bold headings, column widths and a
// filter on a header row. It needs no outside library, only archive/zip, and the files open in Excel, Sheets and Numbers.
package xlsx

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Cell is one value: a number when Num is set, otherwise text. Bold makes a heading.
type Cell struct {
	Text string
	Num  *float64
	Bold bool
}

// S is a text cell, B a bold text cell, N a number cell.
func S(s string) Cell  { return Cell{Text: s} }
func B(s string) Cell  { return Cell{Text: s, Bold: true} }
func N(v float64) Cell { return Cell{Num: &v} }

// Sheet is one worksheet.
type Sheet struct {
	Name      string
	Widths    []float64 // column widths in characters; 0 uses the default
	Rows      [][]Cell
	FilterRow int // 1-based row whose columns get a filter; 0 for none
}

// Workbook is a list of sheets.
type Workbook struct{ Sheets []*Sheet }

func esc(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

// col is the spreadsheet letter of a 0-based column: 0 is A, 26 is AA.
func col(i int) string {
	s := ""
	for i++; i > 0; i = (i - 1) / 26 {
		s = string(rune('A'+(i-1)%26)) + s
	}
	return s
}

// sheetName makes a name Excel accepts: at most 31 characters and none of : \ / ? * [ ].
func sheetName(n string, i int) string {
	n = strings.Map(func(r rune) rune {
		if strings.ContainsRune(`:\/?*[]`, r) {
			return ' '
		}
		return r
	}, n)
	n = strings.TrimSpace(n)
	if n == "" {
		n = fmt.Sprintf("Sheet%d", i)
	}
	if r := []rune(n); len(r) > 31 {
		n = string(r[:31])
	}
	return n
}

// Write writes the workbook to w.
func (wb *Workbook) Write(w io.Writer) error {
	zw := zip.NewWriter(w)
	put := func(name, body string) error {
		f, err := zw.Create(name)
		if err != nil {
			return err
		}
		_, err = io.WriteString(f, body)
		return err
	}
	var ct, rels, sheets strings.Builder
	for i := range wb.Sheets {
		n := i + 1
		fmt.Fprintf(&ct, `<Override PartName="/xl/worksheets/sheet%d.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>`, n)
		fmt.Fprintf(&rels, `<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet%d.xml"/>`, n, n)
		fmt.Fprintf(&sheets, `<sheet name="%s" sheetId="%d" r:id="rId%d"/>`, esc(sheetName(wb.Sheets[i].Name, n)), n, n)
	}
	styleRel := len(wb.Sheets) + 1
	fmt.Fprintf(&rels, `<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>`, styleRel)
	const head = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`
	if err := put("[Content_Types].xml", head+`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/>`+ct.String()+`</Types>`); err != nil {
		return err
	}
	if err := put("_rels/.rels", head+`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>`); err != nil {
		return err
	}
	if err := put("xl/workbook.xml", head+`<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets>`+sheets.String()+`</sheets></workbook>`); err != nil {
		return err
	}
	if err := put("xl/_rels/workbook.xml.rels", head+`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">`+rels.String()+`</Relationships>`); err != nil {
		return err
	}
	// style 0 is normal, style 1 is bold
	if err := put("xl/styles.xml", head+`<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><fonts count="2"><font><sz val="11"/><name val="Calibri"/></font><font><b/><sz val="11"/><name val="Calibri"/></font></fonts><fills count="2"><fill><patternFill patternType="none"/></fill><fill><patternFill patternType="gray125"/></fill></fills><borders count="1"><border><left/><right/><top/><bottom/><diagonal/></border></borders><cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs><cellXfs count="2"><xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0"/><xf numFmtId="0" fontId="1" fillId="0" borderId="0" xfId="0" applyFont="1"/></cellXfs></styleSheet>`); err != nil {
		return err
	}
	for i, sh := range wb.Sheets {
		var b strings.Builder
		b.WriteString(head + `<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">`)
		widest := 0
		for _, r := range sh.Rows {
			widest = max(widest, len(r))
		}
		if len(sh.Widths) > 0 {
			b.WriteString("<cols>")
			for c, wd := range sh.Widths {
				if wd > 0 {
					fmt.Fprintf(&b, `<col min="%d" max="%d" width="%.1f" customWidth="1"/>`, c+1, c+1, wd)
				}
			}
			b.WriteString("</cols>")
		}
		b.WriteString("<sheetData>")
		for ri, r := range sh.Rows {
			fmt.Fprintf(&b, `<row r="%d">`, ri+1)
			for ci, c := range r {
				ref := col(ci) + strconv.Itoa(ri+1)
				st := ""
				if c.Bold {
					st = ` s="1"`
				}
				switch {
				case c.Num != nil:
					fmt.Fprintf(&b, `<c r="%s"%s><v>%s</v></c>`, ref, st, strconv.FormatFloat(*c.Num, 'f', -1, 64))
				case c.Text == "":
					fmt.Fprintf(&b, `<c r="%s"%s/>`, ref, st)
				default:
					fmt.Fprintf(&b, `<c r="%s"%s t="inlineStr"><is><t xml:space="preserve">%s</t></is></c>`, ref, st, esc(c.Text))
				}
			}
			b.WriteString("</row>")
		}
		b.WriteString("</sheetData>")
		if sh.FilterRow > 0 && sh.FilterRow <= len(sh.Rows) && widest > 0 {
			fmt.Fprintf(&b, `<autoFilter ref="A%d:%s%d"/>`, sh.FilterRow, col(widest-1), len(sh.Rows))
		}
		b.WriteString("</worksheet>")
		if err := put(fmt.Sprintf("xl/worksheets/sheet%d.xml", i+1), b.String()); err != nil {
			return err
		}
	}
	return zw.Close()
}
