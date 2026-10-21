package server

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

func TestATripPlaysBackWithBatteryAndDistance(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	t0 := time.Now().Add(-3 * time.Hour).Truncate(time.Second).Unix()
	for i := int64(0); i <= 10; i++ { // ten minutes, a reading a minute, heading north
		srv.S.DB.Exec(`INSERT INTO samples(ts,lat,lon,speed_kph) VALUES(?,?,?,?)`, t0+i*60, 40+float64(i)*0.01, -75.0, 60.0)
		srv.S.DB.Exec(`INSERT INTO trends(ts,soc,speed_kph) VALUES(?,?,?)`, t0+i*60, 80-float64(i), 60.0)
	}
	end := t0 + 600
	tr := &store.Trip{StartTS: t0, Category: "personal", CategorySource: "default"}
	_ = srv.S.CreateTrip(tr)
	tr.EndTS, tr.DistanceKm, tr.Open = &end, 11, false
	_ = srv.S.UpdateTrip(tr)
	code, body := do(t, admin, "GET", fmt.Sprintf("%s/api/trips/%d/replay", ts.URL, tr.ID), "", nil)
	var d struct {
		Points []struct {
			TS  int64
			SoC *float64
			Km  float64
		}
		Source string
	}
	if code != 200 || json.Unmarshal([]byte(body), &d) != nil || len(d.Points) != 11 || d.Source != "car" {
		t.Fatalf("replay: %d %s", code, body)
	}
	last := d.Points[10]
	if *d.Points[0].SoC != 80 || *last.SoC != 70 || last.Km < 10.99 || last.Km > 11.01 || d.Points[5].Km < 5.4 || d.Points[5].Km > 5.6 {
		t.Fatalf("battery and distance along the way: %+v", d.Points)
	}
}

func TestARemoteStartShowsInTheDayView(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	day := time.Now().AddDate(0, 0, -2)
	base := time.Date(day.Year(), day.Month(), day.Day(), 7, 0, 0, 0, srv.defaultLoc()).Unix()
	srv.S.DB.Exec(`CREATE TABLE IF NOT EXISTS remote_starts (id INTEGER PRIMARY KEY, start_ts INTEGER NOT NULL, end_ts INTEGER, start_soc REAL, end_soc REAL, start_kwh REAL, end_kwh REAL, temp_c REAL, restarts INTEGER NOT NULL DEFAULT 0, drove INTEGER NOT NULL DEFAULT 0, last_ts INTEGER, last_left REAL)`)
	srv.S.DB.Exec(`INSERT INTO remote_starts(start_ts,end_ts,start_soc,end_soc,start_kwh,end_kwh,temp_c) VALUES(?,?,80,78,70,68.5,-6)`, base, base+600)
	addDrive(t, srv, time.Unix(base+3600, 0), 20)
	_, body := do(t, admin, "GET", ts.URL+"/api/day?date="+time.Unix(base, 0).In(srv.defaultLoc()).Format("2006-01-02"), "", nil)
	if !strings.Contains(body, `"kind":"remote"`) || !strings.Contains(body, `"tempC":-6`) {
		t.Fatalf("a remote start row: %.400s", body)
	}
}
