package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/settings"
	"github.com/ZachCurry13/cartrak/internal/store"
	"github.com/ZachCurry13/cartrak/internal/weather"
)

func TestTheWindLookupIsOffByDefaultThenGivesATailwindAskingOnlyForACellOnce(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	var asked int32
	var lastQuery string
	wx := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&asked, 1)
		lastQuery = r.URL.RawQuery
		var times, speeds, dirs, gusts []string
		for h := 0; h < 24; h++ {
			times = append(times, fmt.Sprintf(`"%s"`, time.Date(2026, 1, 1, h, 0, 0, 0, time.UTC).Format("2006-01-02T15:04")))
			speeds, dirs, gusts = append(speeds, "25"), append(dirs, "270"), append(gusts, "40") // from the west
		}
		day := r.URL.Query().Get("start_date")
		body := `{"hourly":{"time":[` + strings.ReplaceAll(strings.Join(times, ","), "2026-01-01", day) + `],"wind_speed_10m":[` + strings.Join(speeds, ",") + `],"wind_direction_10m":[` + strings.Join(dirs, ",") + `],"wind_gusts_10m":[` + strings.Join(gusts, ",") + `]}}`
		w.Write([]byte(body))
	}))
	defer wx.Close()
	srv.Weather = &weather.Client{ForecastURL: wx.URL, ArchiveURL: wx.URL}

	fp := func(v float64) *float64 { return &v }
	start := time.Now().Add(-4 * time.Hour).Truncate(time.Second)
	odo, kwh, lon := 9000.0, 85.0, -93.2
	for i := 0; i <= 120; i++ { // eastbound at 100 km/h
		st := &ford.State{TS: start.Add(time.Duration(i*30) * time.Second), OdometerKm: fp(odo), EnergyKwh: fp(kwh), SpeedKph: fp(100), Lat: fp(45.0), Lon: fp(lon), OutsideC: fp(15), Ignition: "ON", Raw: []byte(`{}`)}
		if _, err := srv.S.SaveState("t", st); err != nil {
			t.Fatal(err)
		}
		d := 100.0 / 3600 * 30
		odo += d
		kwh -= d * 0.17
		lon += d / (111.32 * 0.7071) // degrees of longitude at 45 north
	}
	end := start.Add(60 * time.Minute).Unix()
	tr := &store.Trip{StartTS: start.Unix(), EndTS: &end, DistanceKm: 100, StartKwh: fp(85), Category: "personal", CategorySource: "default"}
	if err := srv.S.CreateTrip(tr); err != nil {
		t.Fatal(err)
	}
	tr.EndTS, tr.Open, tr.EndKwh = &end, false, fp(85-17)
	srv.S.UpdateTrip(tr)
	why := func() (headKph *float64, tags []string) {
		code, b := do(t, admin, "GET", fmt.Sprintf("%s/api/trips/%d/why", ts.URL, tr.ID), "", nil)
		var out struct {
			Facts struct {
				Wind *struct{ HeadKph float64 }
			}
			Tags []struct{ Key string }
		}
		if code != 200 || json.Unmarshal([]byte(b), &out) != nil {
			t.Fatalf("%d %s", code, b)
		}
		for _, tg := range out.Tags {
			tags = append(tags, tg.Key)
		}
		if out.Facts.Wind != nil {
			return &out.Facts.Wind.HeadKph, tags
		}
		return nil, tags
	}
	if h, _ := why(); h != nil || atomic.LoadInt32(&asked) != 0 {
		t.Fatalf("off by default: nothing is looked up or sent (asked %d)", asked)
	}
	srv.Cfg.Set(settings.WeatherKey, "on")
	h, _ := why()
	if h == nil || *h > -20 || *h < -26 {
		t.Fatalf("25 km/h from the west for an eastbound car is a tailwind of about 25: %v", h)
	}
	if atomic.LoadInt32(&asked) == 0 {
		t.Fatal("it asked the weather service")
	}
	if !strings.Contains(lastQuery, "latitude=45.0") || strings.Contains(lastQuery, "45.00") {
		t.Fatalf("only a half-degree cell is sent: %s", lastQuery)
	}
	before := atomic.LoadInt32(&asked)
	why()
	if atomic.LoadInt32(&asked) != before {
		t.Fatal("answers are kept: the same trip does not ask again")
	}
}
