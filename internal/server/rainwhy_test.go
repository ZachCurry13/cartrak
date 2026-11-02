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

// A rainy hour of weather for every hour of the day asked about, in the shape Open-Meteo answers.
func rainyDay(day string, mm string) string {
	var times, wind, dir, gust, rain, snow, code []string
	for h := 0; h < 24; h++ {
		times = append(times, fmt.Sprintf(`"%sT%02d:00"`, day, h))
		wind, dir, gust, rain, snow, code = append(wind, "5"), append(dir, "270"), append(gust, "10"), append(rain, mm), append(snow, "0"), append(code, "63")
	}
	j := func(v []string) string { return "[" + strings.Join(v, ",") + "]" }
	return `{"hourly":{"time":` + j(times) + `,"wind_speed_10m":` + j(wind) + `,"wind_direction_10m":` + j(dir) + `,"wind_gusts_10m":` + j(gust) + `,"precipitation":` + j(rain) + `,"snowfall":` + j(snow) + `,"weather_code":` + j(code) + `}}`
}

func TestARainyTripSaysSoNamesRainAsAReasonAndAskedOnceAndAnOldKeptDayIsAskedAgain(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	var asked int32
	wx := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&asked, 1)
		w.Write([]byte(rainyDay(r.URL.Query().Get("start_date"), "4"))) // moderate rain all day
	}))
	defer wx.Close()
	srv.Weather = &weather.Client{ForecastURL: wx.URL, ArchiveURL: wx.URL}
	srv.Cfg.Set(settings.WeatherKey, "on")

	fp := func(v float64) *float64 { return &v }
	start := time.Now().Add(-4 * time.Hour).Truncate(time.Second)
	odo, kwh, lon := 9000.0, 85.0, -93.2
	for i := 0; i <= 120; i++ {
		st := &ford.State{TS: start.Add(time.Duration(i*30) * time.Second), OdometerKm: fp(odo), EnergyKwh: fp(kwh), SpeedKph: fp(100), Lat: fp(45.0), Lon: fp(lon), OutsideC: fp(15), Ignition: "ON", Raw: []byte(`{}`)}
		if _, err := srv.S.SaveState("t", st); err != nil {
			t.Fatal(err)
		}
		d := 100.0 / 3600 * 30
		odo, kwh, lon = odo+d, kwh-d*0.17, lon+d/(111.32*0.7071)
	}
	// Something kept from before rain was asked about, for the same cell and day: it must be asked for again.
	_, _, cell := weather.Cell(45.0, -93.2)
	day := start.UTC().Truncate(24 * time.Hour)
	legacy := `[{"ts":` + fmt.Sprint(day.Unix()) + `,"speedKph":5,"dirDeg":270,"gustKph":10}]`
	if err := srv.S.WeatherPut(cell, day.Format("2006-01-02"), legacy, time.Now().Unix()); err != nil {
		t.Fatal(err)
	}
	end := start.Add(60 * time.Minute).Unix()
	tr := &store.Trip{StartTS: start.Unix(), EndTS: &end, DistanceKm: 100, StartKwh: fp(85), Category: "personal", CategorySource: "default"}
	if err := srv.S.CreateTrip(tr); err != nil {
		t.Fatal(err)
	}
	tr.EndTS, tr.Open, tr.EndKwh = &end, false, fp(85-17)
	srv.S.UpdateTrip(tr)

	why := func() (label string, rainPct float64, tags []string) {
		code, b := do(t, admin, "GET", fmt.Sprintf("%s/api/trips/%d/why", ts.URL, tr.ID), "", nil)
		var out struct {
			Factors []struct {
				Key string
				Pct float64
			}
			Facts struct {
				Weather *struct {
					Label    string
					TotalPct float64
					Learned  bool
				}
			}
			Tags []struct{ Key string }
		}
		if code != 200 || json.Unmarshal([]byte(b), &out) != nil {
			t.Fatalf("%d %s", code, b)
		}
		if out.Facts.Weather != nil {
			label = out.Facts.Weather.Label
			if out.Facts.Weather.Learned {
				t.Fatal("nothing can be learned from one trip")
			}
		}
		for _, f := range out.Factors {
			if f.Key == "rain" {
				rainPct = f.Pct
			}
		}
		for _, tg := range out.Tags {
			tags = append(tags, tg.Key)
		}
		return
	}
	label, pct, tags := why()
	if label != "Rain" || pct != 8 {
		t.Fatalf("moderate rain all the way is worth the fixed 8%%: %q %v %v", label, pct, tags)
	}
	has := false
	for _, k := range tags {
		has = has || k == "rain"
	}
	if !has {
		t.Fatalf("a rain tag: %v", tags)
	}
	if atomic.LoadInt32(&asked) == 0 {
		t.Fatal("the day kept before rain was asked about is asked for again")
	}
	before := atomic.LoadInt32(&asked)
	why()
	if atomic.LoadInt32(&asked) != before {
		t.Fatal("and then kept")
	}
}

func TestTheBackgroundBackfillFillsInRecentTripsOnlyWhenTheLookupIsOn(t *testing.T) {
	_, srv := healthServer(t, fakeProv{}, false, false)
	var asked int32
	wx := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&asked, 1)
		w.Write([]byte(rainyDay(r.URL.Query().Get("start_date"), "0")))
	}))
	defer wx.Close()
	srv.Weather = &weather.Client{ForecastURL: wx.URL, ArchiveURL: wx.URL}
	fp := func(v float64) *float64 { return &v }
	start := time.Now().Add(-48 * time.Hour).Truncate(time.Second)
	end := start.Add(30 * time.Minute).Unix()
	tr := &store.Trip{StartTS: start.Unix(), EndTS: &end, DistanceKm: 20, StartLat: fp(45.0), StartLon: fp(-93.2), StartKwh: fp(80), Category: "personal", CategorySource: "default"}
	if err := srv.S.CreateTrip(tr); err != nil {
		t.Fatal(err)
	}
	tr.EndTS, tr.Open, tr.EndKwh = &end, false, fp(76)
	srv.S.UpdateTrip(tr)

	ctx := t.Context()
	srv.backfillWeather(ctx)
	if atomic.LoadInt32(&asked) != 0 {
		t.Fatal("off by default: nothing is asked")
	}
	srv.Cfg.Set(settings.WeatherKey, "on")
	srv.backfillWeather(ctx)
	if atomic.LoadInt32(&asked) != 1 {
		t.Fatalf("one day for one trip: asked %d", asked)
	}
	srv.backfillWeather(ctx)
	if atomic.LoadInt32(&asked) != 1 {
		t.Fatal("a day already kept is not asked again")
	}
}

func TestTheRainEffectIsLearnedFromTheOwnersWetAndDryTrips(t *testing.T) {
	_, srv := healthServer(t, fakeProv{}, false, false)
	fp := func(v float64) *float64 { return &v }
	// A drive of readings so there is a usual efficiency to compare with: 5.88 km per kWh.
	start := time.Now().Add(-3 * time.Hour).Truncate(time.Second)
	odo, kwh := 9000.0, 85.0
	for i := 0; i <= 120; i++ {
		st := &ford.State{TS: start.Add(time.Duration(i*30) * time.Second), OdometerKm: fp(odo), EnergyKwh: fp(kwh), SpeedKph: fp(100), Lat: fp(45.0), Lon: fp(-93.2), Ignition: "ON", Raw: []byte(`{}`)}
		if _, err := srv.S.SaveState("t", st); err != nil {
			t.Fatal(err)
		}
		d := 100.0 / 3600 * 30
		odo, kwh = odo+d, kwh-d*0.17
	}
	_, _, cell := weather.Cell(45.0, -93.2)
	for i := 0; i < 16; i++ {
		day := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, -2-i)
		wet := i%2 == 0
		mm, perKm := 0.0, 0.17
		if wet {
			mm, perKm = 2.0, 0.19 // rainy days cost about 12% more per kilometer
		}
		var hs []weather.Hour
		for h := 0; h < 24; h++ {
			hs = append(hs, weather.Hour{TS: day.Add(time.Duration(h) * time.Hour).Unix(), V: 2, Wx: true, RainMm: mm})
		}
		b, _ := json.Marshal(hs)
		if err := srv.S.WeatherPut(cell, day.Format("2006-01-02"), string(b), time.Now().Unix()); err != nil {
			t.Fatal(err)
		}
		s0 := day.Add(12 * time.Hour)
		end := s0.Add(30 * time.Minute).Unix()
		tr := &store.Trip{StartTS: s0.Unix(), EndTS: &end, DistanceKm: 20, StartLat: fp(45.0), StartLon: fp(-93.2), StartKwh: fp(80), Category: "personal", CategorySource: "default"}
		if err := srv.S.CreateTrip(tr); err != nil {
			t.Fatal(err)
		}
		tr.EndTS, tr.Open, tr.EndKwh = &end, false, fp(80-20*perKm)
		srv.S.UpdateTrip(tr)
	}
	l, ok := srv.learnedRain()
	if !ok || l.WetTrips != 8 || l.DryTrips != 8 || l.EffectPct < 10 || l.EffectPct > 14 {
		t.Fatalf("about 12%% more energy in the rain, measured from 8 wet and 8 dry trips: %+v ok=%v", l, ok)
	}
}
