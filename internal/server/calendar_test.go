package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/settings"
	"github.com/ZachCurry13/cartrak/internal/store"
)

// icsFor is a calendar with one appointment on a day: 10:00 to 11:00 in the car's zone (written without a zone, so it is read in it).
func icsFor(day time.Time, title, location string) string {
	d := day.Format("20060102")
	return "BEGIN:VCALENDAR\r\nBEGIN:VEVENT\r\nUID:a\r\nSUMMARY:" + title + "\r\nLOCATION:" + location + "\r\nDTSTART:" + d + "T100000\r\nDTEND:" + d + "T110000\r\nEND:VEVENT\r\n" +
		"BEGIN:VEVENT\r\nUID:b\r\nSUMMARY:Day off\r\nDTSTART;VALUE=DATE:" + d + "\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"
}

func calendarServer(t *testing.T, ics func() string) (*httptest.Server, *Server) {
	cal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(ics())) }))
	t.Cleanup(cal.Close)
	ts, srv := healthServer(t, fakeProv{}, false, false)
	srv.Cfg.Set(settings.CalendarURLKey, cal.URL+"/private/basic.ics")
	if err := srv.S.SetKV("company:777", `{"name":"Acme Widgets","address":"1 Main St","lat":44.95,"lon":-93.25}`); err != nil {
		t.Fatal(err)
	}
	return ts, srv
}

func TestTodaysAppointmentsFindTheirPlaceAmongTheAccounts(t *testing.T) {
	now := time.Now()
	ts, srv := calendarServer(t, func() string { return icsFor(now.In(time.Local), "Visit Acme Widgets", "") })
	srv.Cfg.Set("CARTRAK_TZ", time.Local.String())
	admin := adminClient(t, ts.URL)
	code, body := do(t, admin, "GET", ts.URL+"/api/calendar/day?date="+now.In(srv.Cfg.Location()).Format("2006-01-02"), "", nil)
	var r struct {
		Connected bool
		Events    []struct {
			Title, From string
			AllDay      bool
			Place       *struct{ Address string }
		}
	}
	if code != 200 || json.Unmarshal([]byte(body), &r) != nil || !r.Connected || len(r.Events) != 2 {
		t.Fatalf("%d %s", code, body)
	}
	if !r.Events[0].AllDay || r.Events[0].Place != nil {
		t.Fatalf("an all-day appointment has no place: %+v", r.Events[0])
	}
	if e := r.Events[1]; e.From != "account" || e.Place == nil || e.Place.Address != "1 Main St" {
		t.Fatalf("found among the accounts by its title: %+v", e)
	}
	if strings.Contains(body, "/private/") {
		t.Fatal("the calendar address must never reach the browser")
	}
	if c, _ := do(t, &http.Client{}, "GET", ts.URL+"/api/calendar/day", "", nil); c != 401 {
		t.Fatalf("appointments are for the admin only, got %d", c)
	}
}

func TestATripThatEndsAtAnAppointmentIsNamedAfterIt(t *testing.T) {
	loc := time.Local
	day := time.Date(2026, 11, 10, 0, 0, 0, 0, loc)
	now := day.Add(12 * time.Hour)
	_, srv := calendarServer(t, func() string { return icsFor(day, "Visit Acme Widgets", "") })
	srv.Cfg.Set("CARTRAK_TZ", loc.String())
	srv.Cfg.Set(settings.CalendarWorkKey, "work")
	at := time.Date(day.Year(), day.Month(), day.Day(), 9, 55, 0, 0, loc) // arrives five minutes before 10:00
	lat, lon, far := 44.9501, -93.2501, 45.5
	mk := func(endLat float64) int64 {
		tr := &store.Trip{StartTS: at.Add(-30 * time.Minute).Unix(), Category: "personal", CategorySource: "default"}
		if err := srv.S.CreateTrip(tr); err != nil {
			t.Fatal(err)
		}
		end := at.Unix()
		tr.EndTS, tr.EndLat, tr.EndLon, tr.Open = &end, &endLat, &lon, false
		if err := srv.S.UpdateTrip(tr); err != nil {
			t.Fatal(err)
		}
		return tr.ID
	}
	near, other := mk(lat), mk(far)
	if n := srv.nameTripsFromCalendar(context.Background(), now); n != 1 {
		t.Fatalf("one trip named, got %d", n)
	}
	a, _ := srv.S.GetTrip(near)
	b, _ := srv.S.GetTrip(other)
	if a.Notes != "Visit Acme Widgets" || a.Category != "work" || b.Notes != "" {
		t.Fatalf("named and counted as work: %+v; the other left alone: %+v", a, b)
	}
}

func TestTheEveningHeadsUpNamesTheFirstAppointmentAndTheWarningAtIt(t *testing.T) {
	loc := time.Local
	evening := time.Date(2026, 11, 15, 19, 5, 0, 0, loc)
	_, srv := calendarServer(t, func() string { return icsFor(evening.AddDate(0, 0, 1), "Visit Acme Widgets", "") })
	srv.Cfg.Set("CARTRAK_TZ", loc.String())
	srv.Cfg.Set(settings.WeatherAlertsKey, "on")
	wx := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"features":[{"properties":{"id":"w1","event":"Winter Storm Warning","severity":"Severe","status":"Actual","messageType":"Alert","headline":"Heavy snow"}}]}`)
	}))
	defer wx.Close()
	srv.NWSURL = wx.URL
	srv.dayAheadOnce(context.Background(), evening)
	srv.dayAheadOnce(context.Background(), evening.Add(10*time.Minute))
	got, _ := srv.S.RecentAlerts(10)
	if len(got) != 1 || got[0].Kind != "day_ahead" || !strings.Contains(got[0].Body, "First at 10:00 am, Visit Acme Widgets.") || !strings.Contains(got[0].Body, "Winter Storm Warning at Acme Widgets") {
		t.Fatalf("one heads-up an evening: %+v", got)
	}
}

func TestARoadWarningWhereTheCarIsIsSentOnce(t *testing.T) {
	_, srv := healthServer(t, fakeProv{}, false, false)
	srv.Cfg.Set(settings.WeatherAlertsKey, "on")
	asked := 0
	wx := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked++
		fmt.Fprint(w, `{"features":[{"properties":{"id":"w1","event":"Ice Storm Warning","severity":"Severe","status":"Actual","messageType":"Alert","headline":"Ice on roads"}},
 {"properties":{"id":"h1","event":"Heat Advisory","severity":"Moderate","status":"Actual","messageType":"Alert"}}]}`)
	}))
	defer wx.Close()
	srv.NWSURL = wx.URL
	lat, lon := 44.9, -93.2
	if _, err := srv.S.SaveState("demo", &ford.State{TS: time.Now(), Lat: &lat, Lon: &lon, Extra: map[string]any{}}); err != nil {
		t.Fatal(err)
	}
	srv.weatherWatch(context.Background(), time.Now())
	srv.weatherWatch(context.Background(), time.Now())
	got, _ := srv.S.RecentAlerts(10)
	if len(got) != 1 || got[0].Title != "Ice Storm Warning" || asked != 1 {
		t.Fatalf("the road warning once, the heat advisory not at all, the service asked once: %+v (%d)", got, asked)
	}
}

func TestLiveChargersNeedTheKeyAndAreAskedOnceInFiveMinutes(t *testing.T) {
	asked := 0
	g := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked++
		w.Write([]byte(`{"places":[{"displayName":{"text":"Supercharger"},"location":{"latitude":45,"longitude":-93},"evChargeOptions":{"connectorAggregation":[{"type":"EV_CONNECTOR_TYPE_TESLA","maxChargeRateKw":250,"count":8,"availableCount":5}]}}]}`))
	}))
	defer g.Close()
	ts, srv := healthServer(t, fakeProv{}, false, false)
	srv.PlacesURL = g.URL
	admin := adminClient(t, ts.URL)
	if c, _ := do(t, admin, "GET", ts.URL+"/api/chargers/live?lat=45.123&lon=-93.456", "", nil); c != 400 {
		t.Fatalf("no key: %d", c)
	}
	srv.Cfg.Set(settings.GooglePlacesKeyKey, "secret-key")
	for i := 0; i < 2; i++ {
		code, body := do(t, admin, "GET", ts.URL+"/api/chargers/live?lat=45.123&lon=-93.456", "", nil)
		if code != 200 || !strings.Contains(body, `"available":5`) || strings.Contains(body, "secret-key") {
			t.Fatalf("%d %s", code, body)
		}
	}
	if asked != 1 {
		t.Fatalf("asked %d times", asked)
	}
}
