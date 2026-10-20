package xlsx

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"
	"strings"
	"testing"
)

func TestAWorkbookIsWellFormedAndKeepsItsCells(t *testing.T) {
	wb := &Workbook{Sheets: []*Sheet{
		{Name: "Trips: 2026/Q1?", Widths: []float64{12, 0, 20}, FilterRow: 2, Rows: [][]Cell{
			{B("Summary")},
			{B("Date"), B("Miles"), B("From")},
			{S("2026-10-13"), N(6.5), S(`Home & "Cabin" <north>`)},
			{S(""), N(0), S("x")},
		}},
		{Name: "", Rows: [][]Cell{{S("a")}}},
	}}
	var buf bytes.Buffer
	if err := wb.Write(&buf); err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{}
	for _, f := range zr.File {
		rc, _ := f.Open()
		b, _ := io.ReadAll(rc)
		rc.Close()
		files[f.Name] = string(b)
		var v any
		if err := xml.Unmarshal(b, &v); err != nil && !strings.HasSuffix(f.Name, ".rels") {
			t.Fatalf("%s is not well-formed XML: %v", f.Name, err)
		}
	}
	for _, want := range []string{"[Content_Types].xml", "_rels/.rels", "xl/workbook.xml", "xl/styles.xml", "xl/worksheets/sheet1.xml", "xl/worksheets/sheet2.xml"} {
		if files[want] == "" {
			t.Fatalf("missing %s", want)
		}
	}
	s1 := files["xl/worksheets/sheet1.xml"]
	for _, want := range []string{`<c r="B3"><v>6.5</v></c>`, `Home &amp; &#34;Cabin&#34; &lt;north&gt;`, `<autoFilter ref="A2:C4"/>`, `<c r="A2" s="1" t="inlineStr">`} {
		if !strings.Contains(s1, want) {
			t.Fatalf("sheet 1 lacks %q:\n%s", want, s1)
		}
	}
	if !strings.Contains(files["xl/workbook.xml"], `name="Trips  2026 Q1 "`) && !strings.Contains(files["xl/workbook.xml"], `name="Trips  2026 Q1"`) {
		t.Fatalf("sheet names are cleaned: %s", files["xl/workbook.xml"])
	}
	if col(0) != "A" || col(25) != "Z" || col(26) != "AA" || col(27) != "AB" {
		t.Fatal("column letters")
	}
}
