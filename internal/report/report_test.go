package report

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"
	"time"
)

func trip(day int, cat string, km float64) Trip {
	s := time.Date(2026, 3, day, 15, 26, 0, 0, time.UTC)
	return Trip{ID: int64(day), Start: s, End: s.Add(48 * time.Minute), Zone: "America/Chicago", Category: cat, DistanceKm: km, From: "Home", To: "Cabin", Method: "Car"}
}

func TestTheStandardReportHasTheSampleColumnsAndWorksOutValues(t *testing.T) {
	o := Options{Labels: map[string]string{"work": "Work", "personal": "Personal"}, Rates: map[string]float64{"work": 0.76}, Vehicle: "My car", Summary: true, Person: "Ann", Range: "March 1, 2026 - March 31, 2026", Purpose: "All"}
	ts := []Trip{trip(5, "personal", 16.0934), trip(3, "work", 16.0934)}
	rows, total := Sum(ts, o)
	if len(rows) != 2 || total.Trips != 2 || total.Distance < 19.99 || total.Distance > 20.01 || total.Value < 7.59 || total.Value > 7.61 {
		t.Fatalf("20 miles of which 10 are work at 0.76: %+v %+v", rows, total)
	}
	var buf bytes.Buffer
	if err := CSV(&buf, ts, o); err != nil {
		t.Fatal(err)
	}
	recs, _ := csv.NewReader(&buf).ReadAll()
	want := "Value,Miles,Mileage Rate,From,To,Date,Purpose,Track Method,Notes,Vehicle,Time Started,Time Ended,Duration (hours),Time Zone"
	if strings.Join(recs[0], ",") != want {
		t.Fatalf("the standard columns: %v", recs[0])
	}
	if recs[1][5] != "2026-03-03" || recs[1][6] != "Work" || recs[1][0] != "7.6" || recs[1][1] != "10" || recs[1][10] != "03:26 PM" || recs[1][12] != "0.8" {
		t.Fatalf("the earliest trip first, with its value: %v", recs[1])
	}
	// the workbook has a summary above the table and a second sheet of totals
	wb := Workbook(ts, o)
	if len(wb.Sheets) != 2 || wb.Sheets[1].Name != "Totals By Label" || wb.Sheets[0].FilterRow != 14 {
		t.Fatalf("sheets: %d filter row %d", len(wb.Sheets), wb.Sheets[0].FilterRow)
	}
}

func TestColumnsCanBeChosenRenamedAndExtended(t *testing.T) {
	o := Options{Metric: true, Labels: map[string]string{"work": "Work"}, Columns: []Pick{{Key: "date", Label: "Trip date"}, {Key: "distance"}, {Key: "nonsense"}, {Key: "purpose", Label: "Reason"}},
		Fixed: []Fixed{{Label: "Company", Value: "Acme"}}}
	var buf bytes.Buffer
	_ = CSV(&buf, []Trip{trip(3, "work", 12.5)}, o)
	recs, _ := csv.NewReader(&buf).ReadAll()
	if strings.Join(recs[0], ",") != "Trip date,Kilometres,Reason,Company" || strings.Join(recs[1], ",") != "2026-03-03,12.5,Work,Acme" {
		t.Fatalf("custom columns: %v", recs)
	}
	// text that looks like a formula is made harmless in a CSV
	bad := trip(4, "work", 1)
	bad.Notes = "=HYPERLINK(\"x\")"
	buf.Reset()
	_ = CSV(&buf, []Trip{bad}, Options{Columns: []Pick{{Key: "notes"}}})
	if !strings.Contains(buf.String(), "'=HYPERLINK") {
		t.Fatalf("formula text must be defused: %s", buf.String())
	}
}

func TestFunColumnsShowEnergyCostEfficiencyInTheChosenUnitAndMPGe(t *testing.T) {
	kwh := 5.0
	tr := trip(3, "work", 16.0934) // ten miles in 48 minutes on 5 kWh
	tr.EnergyKwh = &kwh
	cols := []Pick{{Key: "energy"}, {Key: "efficiency"}, {Key: "energycost"}, {Key: "costper"}, {Key: "mpge"}, {Key: "avgspeed"}}
	run := func(o Options) []string {
		o.Columns = cols
		var buf bytes.Buffer
		_ = CSV(&buf, []Trip{tr}, o)
		recs, _ := csv.NewReader(&buf).ReadAll()
		return append(recs[0], recs[1]...)
	}
	got := run(Options{KwhPrice: 0.20})
	if strings.Join(got, "|") != "Energy Used (kWh)|Efficiency|Energy Cost|Cost per Mile|MPGe|Average Speed (mph)|5|2.00 mi/kWh|1|0.1|67|12" {
		t.Fatalf("miles per kWh: %v", got)
	}
	if got := run(Options{EffUnit: "wh", KwhPrice: 0.20}); got[7] != "500 Wh/mi" {
		t.Fatalf("watt-hours per mile: %v", got)
	}
	if got := run(Options{EffUnit: "pct", Capacity: 80, KwhPrice: 0.20}); got[7] != "1.60 mi/%" {
		t.Fatalf("miles per percent: %v", got)
	}
	m := run(Options{Metric: true, KwhPrice: 0.20})
	if m[7] != "31.1 kWh/100 km" || m[3] != "Cost per Kilometre" || m[9] != "0.062" {
		t.Fatalf("kilometres: %v", m)
	}
	if got := run(Options{}); got[8] != "" || got[9] != "" {
		t.Fatalf("no price known, no cost: %v", got)
	}
}
