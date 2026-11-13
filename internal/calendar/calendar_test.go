package calendar

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

var chicago, _ = time.LoadLocation("America/Chicago")

const sample = "BEGIN:VCALENDAR\r\nVERSION:2.0\r\n" +
	"BEGIN:VEVENT\r\nUID:one\r\nSUMMARY:Visit Acme Widgets\\, Inc.\r\nLOCATION:1 Main St\\, Townsville\\, MN\r\nDTSTART;TZID=America/Chicago:20261116T090000\r\nDTEND;TZID=America/Chicago:20261116T100000\r\nEND:VEVENT\r\n" +
	"BEGIN:VEVENT\r\nUID:weekly\r\nSUMMARY:Lakeshore Supply\r\nLOCATION:Lakeshore Supply\r\nDTSTART;TZID=Central Standard Time:20261102T130000\r\nDURATION:PT45M\r\n" +
	"RRULE:FREQ=WEEKLY;INTERVAL=2;BYDAY=MO,WE;UNTIL=20261231T235959Z\r\nEXDATE;TZID=America/Chicago:20261118T130000\r\nEND:VEVENT\r\n" +
	"BEGIN:VEVENT\r\nUID:weekly\r\nRECURRENCE-ID;TZID=America/Chicago:20261116T130000\r\nSUMMARY:Lakeshore Supply (moved)\r\nLOCATION:Lakeshore Supply\r\nDTSTART;TZID=America/Chicago:20261116T150000\r\nDTEND;TZID=America/Chicago:20261116T153000\r\nEND:VEVENT\r\n" +
	"BEGIN:VEVENT\r\nUID:gone\r\nSUMMARY:Cancelled\r\nSTATUS:CANCELLED\r\nDTSTART:20261116T150000Z\r\nDTEND:20261116T160000Z\r\nEND:VEVENT\r\n" +
	"BEGIN:VEVENT\r\nUID:allday\r\nSUMMARY:Holiday\r\nDTSTART;VALUE=DATE:20261116\r\nEND:VEVENT\r\n" +
	"BEGIN:VEVENT\r\nUID:long\r\nSUMMARY:A title that goes on\r\n  and on\r\nDTSTART:20261116T200000Z\r\nDTEND:20261116T210000Z\r\nEND:VEVENT\r\n" +
	"END:VCALENDAR\r\n"

func day(y int, m time.Month, d int) (time.Time, time.Time) {
	a := time.Date(y, m, d, 0, 0, 0, 0, chicago)
	return a, a.AddDate(0, 0, 1)
}

func titles(o []Occurrence) string {
	var t []string
	for _, x := range o {
		t = append(t, x.Start.In(chicago).Format("15:04")+" "+x.Title)
	}
	return strings.Join(t, "; ")
}

func TestADayListsSingleRepeatingMovedAndAllDayAppointments(t *testing.T) {
	ev := Parse(sample, chicago)
	got := titles(on(ev, 2026, 11, 16))
	want := "00:00 Holiday; 09:00 Visit Acme Widgets, Inc.; 14:00 A title that goes on and on; 15:00 Lakeshore Supply (moved)"
	if got != want {
		t.Fatalf("\n got %s\nwant %s", got, want)
	}
	o := on(ev, 2026, 11, 16)
	if o[1].Location != "1 Main St, Townsville, MN" || o[1].End.Sub(o[1].Start) != time.Hour {
		t.Fatalf("location and length: %+v", o[1])
	}
}

func TestEveryOtherWeekWithDaysAnEndAndADateTakenOut(t *testing.T) {
	ev := Parse(sample, chicago)
	var got []string
	for d := 1; d <= 30; d++ {
		for _, o := range on(ev, 2026, 11, d) {
			if o.UID == "weekly" {
				got = append(got, o.Start.In(chicago).Format("Jan 2 15:04"))
			}
		}
	}
	// weeks of Nov 2 and Nov 16 and Nov 30; Nov 18 taken out; Nov 16 moved to 15:00
	want := "Nov 2 13:00, Nov 4 13:00, Nov 16 15:00, Nov 30 13:00"
	if strings.Join(got, ", ") != want {
		t.Fatalf("\n got %s\nwant %s", strings.Join(got, ", "), want)
	}
	if on(ev, 2027, 1, 11) != nil {
		t.Fatal("the rule ended at the end of the year")
	}
}

func TestMonthlyNthWeekdayDailyCountAndYearly(t *testing.T) {
	mk := func(rule string) []Event {
		return []Event{{UID: "x", Title: "x", Start: time.Date(2026, 1, 13, 9, 0, 0, 0, chicago), End: time.Date(2026, 1, 13, 10, 0, 0, 0, chicago), Rule: rule}}
	}
	if len(on(mk("FREQ=MONTHLY;BYDAY=2TU"), 2026, 11, 10)) != 1 || len(on(mk("FREQ=MONTHLY;BYDAY=2TU"), 2026, 11, 17)) != 0 {
		t.Fatal("the second Tuesday of November is the 10th")
	}
	if len(on(mk("FREQ=MONTHLY;BYDAY=-1FR"), 2026, 11, 27)) != 1 {
		t.Fatal("the last Friday of November is the 27th")
	}
	if len(on(mk("FREQ=DAILY;COUNT=3"), 2026, 1, 15)) != 1 || len(on(mk("FREQ=DAILY;COUNT=3"), 2026, 1, 16)) != 0 {
		t.Fatal("three days from the 13th end on the 15th")
	}
	if len(on(mk("FREQ=YEARLY"), 2027, 1, 13)) != 1 {
		t.Fatal("yearly")
	}
	if len(on(mk("FREQ=MONTHLY"), 2026, 11, 13)) != 1 {
		t.Fatal("monthly by date")
	}
}

func TestFetchTakesWebcalAndNeverShowsTheAddress(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/secret-abc/basic.ics" {
			w.Write([]byte(sample))
			return
		}
		if r.URL.Path == "/page" {
			w.Write([]byte("<html>calendar</html>"))
			return
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()
	if s, err := Fetch(context.Background(), nil, srv.URL+"/secret-abc/basic.ics"); err != nil || !strings.Contains(s, "VEVENT") {
		t.Fatal(err)
	}
	if _, err := Fetch(context.Background(), nil, srv.URL+"/secret-abc/gone.ics"); err == nil || strings.Contains(err.Error(), "secret-abc") {
		t.Fatalf("refused, without the address: %v", err)
	}
	if _, err := Fetch(context.Background(), nil, srv.URL+"/page"); err == nil || !strings.Contains(err.Error(), "not a calendar") {
		t.Fatalf("a web page is not a calendar: %v", err)
	}
	if _, err := Fetch(context.Background(), nil, "ftp://x"); err == nil {
		t.Fatal("only web addresses")
	}
	webcal := "webcal://" + strings.TrimPrefix(srv.URL, "http://") + "/secret-abc/basic.ics"
	if _, err := Fetch(context.Background(), nil, webcal); err == nil || strings.Contains(err.Error(), "secret-abc") {
		t.Fatalf("webcal becomes https (the test server is http, so it fails) without showing the address: %v", err)
	}
}

func on(ev []Event, y int, m time.Month, d int) []Occurrence {
	a, b := day(y, m, d)
	return Between(ev, a, b)
}
