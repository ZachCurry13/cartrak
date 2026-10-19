package weather

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestACellIsHalfADegreeSoNoExactPositionIsEverSent(t *testing.T) {
	lat, lon, key := Cell(44.9778, -93.2650)
	if lat != 45.0 || lon != -93.5 || key != "45.0,-93.5" {
		t.Fatalf("%v %v %q", lat, lon, key)
	}
}

func TestDayAsksForOneDayOfWindForACellAndParsesTheHours(t *testing.T) {
	var got string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.RawQuery
		w.Write([]byte(`{"hourly":{"time":["2026-10-18T00:00","2026-10-18T01:00"],"wind_speed_10m":[12.5,20],"wind_direction_10m":[270,280],"wind_gusts_10m":[25,40]}}`))
	}))
	defer ts.Close()
	c := &Client{ForecastURL: ts.URL, ArchiveURL: ts.URL}
	hours, err := c.Day(context.Background(), 45.0, -93.5, time.Date(2026, 10, 18, 5, 0, 0, 0, time.UTC).AddDate(0, 0, 0))
	if err != nil || len(hours) != 2 || hours[1].SpeedKm != 20 || hours[1].DirDeg != 280 || hours[1].GustKm != 40 {
		t.Fatalf("%v %+v", err, hours)
	}
	for _, want := range []string{"latitude=45.0", "longitude=-93.5", "wind_speed_10m", "start_date=2026-10-18", "end_date=2026-10-18"} {
		if !strings.Contains(got, want) {
			t.Fatalf("the request must carry %q: %s", want, got)
		}
	}
	if strings.Contains(got, "44.97") {
		t.Fatalf("only the cell, never the position: %s", got)
	}
}

func TestAFailingServiceIsAnError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "no", 500) }))
	defer ts.Close()
	if _, err := (&Client{ForecastURL: ts.URL}).Day(context.Background(), 45, -93, time.Now()); err == nil {
		t.Fatal("a 500 is an error")
	}
}
