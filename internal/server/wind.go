package server

import (
	"context"
	"encoding/json"
	"sort"
	"time"

	"github.com/ZachCurry13/cartrak/internal/weather"
	"github.com/ZachCurry13/cartrak/internal/windcalc"
)

// maxWeatherFetches limits how many days of weather one trip may ask the weather service for in one go. A long road trip
// crosses many cells; the rest are fetched the next time the trip is opened.
const maxWeatherFetches = 14

// windPoints are the moving readings of a trip that have a position.
type windPoint struct {
	ts       int64
	lat, lon float64
	speed    float64
}

// hourLook finds the weather hour nearest a place and time, when the lookup has one.
type hourLook func(lat, lon float64, ts int64) (weather.Hour, bool)

func (s *Server) weatherClient() *weather.Client {
	if s.Weather != nil {
		return s.Weather
	}
	return &weather.Client{}
}

// weatherDay is the hours of one cell and UTC day: from what was kept when it already has rain and snow in it, otherwise asked
// for and kept. An hour kept before rain was asked about is returned when the service cannot be reached, so wind still works.
func (s *Server) weatherDay(ctx context.Context, client *weather.Client, lat, lon float64, day time.Time, mayFetch bool) ([]weather.Hour, bool) {
	_, _, cell := weather.Cell(lat, lon)
	d := day.Format("2006-01-02")
	var old []weather.Hour
	if body := s.S.WeatherGet(cell, d); body != "" {
		var hs []weather.Hour
		if json.Unmarshal([]byte(body), &hs) == nil && len(hs) > 0 {
			if hs[0].V >= 2 {
				return hs, true
			}
			old = hs
		}
	}
	if !mayFetch || ctx.Err() != nil {
		return old, false
	}
	clat, clon, _ := weather.Cell(lat, lon)
	hs, err := client.Day(ctx, clat, clon, day)
	if err != nil || len(hs) == 0 {
		return old, false
	}
	if b, err := json.Marshal(hs); err == nil {
		_ = s.S.WeatherPut(cell, d, string(b), time.Now().Unix())
	}
	return hs, true
}

// weatherLook gets the weather for every cell and day a trip's moving readings touch (asking the optional lookup for what is
// not kept yet) and returns a finder over them, or false when the lookup is off or there is nothing to look up.
func (s *Server) weatherLook(pts []windPoint) (hourLook, bool) {
	if s.Cfg == nil || !s.Cfg.WeatherLookup() || len(pts) < 2 {
		return nil, false
	}
	type need struct {
		lat, lon float64
		day      time.Time
	}
	needs := map[string]need{}
	for _, p := range pts {
		if p.speed < 3 {
			continue
		}
		clat, clon, cell := weather.Cell(p.lat, p.lon)
		day := time.Unix(p.ts, 0).UTC().Truncate(24 * time.Hour)
		needs[cell+"|"+day.Format("2006-01-02")] = need{clat, clon, day}
	}
	keys := make([]string, 0, len(needs))
	for k := range needs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	hours := map[string][]weather.Hour{}
	client := s.weatherClient()
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	fetched := 0
	for _, k := range keys {
		n := needs[k]
		_, _, cell := weather.Cell(n.lat, n.lon)
		_, cached := s.weatherCached(cell, n.day)
		may := !cached && fetched < maxWeatherFetches
		if may {
			fetched++
		}
		if hs, _ := s.weatherDay(ctx, client, n.lat, n.lon, n.day, may); len(hs) > 0 {
			hours[k] = hs
		}
	}
	if len(hours) == 0 {
		return nil, false
	}
	return func(lat, lon float64, ts int64) (weather.Hour, bool) {
		_, _, cell := weather.Cell(lat, lon)
		hs := hours[cell+"|"+time.Unix(ts, 0).UTC().Truncate(24*time.Hour).Format("2006-01-02")]
		if len(hs) == 0 {
			return weather.Hour{}, false
		}
		best := hs[0]
		for _, h := range hs {
			if abs64(h.TS-ts) < abs64(best.TS-ts) {
				best = h
			}
		}
		return best, true
	}, true
}

// weatherCached says whether the kept hours for a cell and day were asked for with rain and snow included.
func (s *Server) weatherCached(cell string, day time.Time) ([]weather.Hour, bool) {
	body := s.S.WeatherGet(cell, day.Format("2006-01-02"))
	if body == "" {
		return nil, false
	}
	var hs []weather.Hour
	if json.Unmarshal([]byte(body), &hs) != nil || len(hs) == 0 {
		return nil, false
	}
	return hs, hs[0].V >= 2
}

// windFor works out the wind met on a trip from the weather found for it, or false when there is none or not enough of it.
func (s *Server) windFor(pts []windPoint, look hourLook) (windcalc.Stats, bool) {
	if look == nil {
		return windcalc.Stats{}, false
	}
	wp := make([]windcalc.Point, 0, len(pts))
	for _, p := range pts {
		wp = append(wp, windcalc.Point{TS: p.ts, Lat: p.lat, Lon: p.lon, Speed: p.speed})
	}
	return windcalc.Estimate(wp, windcalc.Lookup(look))
}

func abs64(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}
