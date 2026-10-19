package server

import (
	"context"
	"encoding/json"
	"sort"
	"time"

	"github.com/ZachCurry13/cartrak/internal/weather"
	"github.com/ZachCurry13/cartrak/internal/windcalc"
)

// maxWeatherFetches limits how many days of wind one trip may ask the weather service for in one go. A long road trip
// crosses many cells; the rest are fetched the next time the trip is opened.
const maxWeatherFetches = 14

// windPoints are the moving readings of a trip that have a position.
type windPoint struct {
	ts       int64
	lat, lon float64
	speed    float64
}

// windFor works out the wind met on a trip from the optional weather lookup, or false when the lookup is off, the
// service could not be reached, or there is not enough data. Answers are kept, so each cell and day is asked for once.
func (s *Server) windFor(pts []windPoint) (windcalc.Stats, bool) {
	if s.Cfg == nil || !s.Cfg.WeatherLookup() || len(pts) < 2 {
		return windcalc.Stats{}, false
	}
	type need struct {
		lat, lon float64
		day      time.Time
	}
	needs := map[string]need{}
	for _, p := range pts {
		if p.speed < 30 {
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
	client := s.Weather
	if client == nil {
		client = &weather.Client{}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	fetched := 0
	for _, k := range keys {
		n := needs[k]
		_, _, cell := weather.Cell(n.lat, n.lon)
		day := n.day.Format("2006-01-02")
		if body := s.S.WeatherGet(cell, day); body != "" {
			var hs []weather.Hour
			if json.Unmarshal([]byte(body), &hs) == nil {
				hours[k] = hs
				continue
			}
		}
		if fetched >= maxWeatherFetches || ctx.Err() != nil {
			continue
		}
		fetched++
		hs, err := client.Day(ctx, n.lat, n.lon, n.day)
		if err != nil || len(hs) == 0 {
			continue
		}
		if b, err := json.Marshal(hs); err == nil {
			_ = s.S.WeatherPut(cell, day, string(b), time.Now().Unix())
		}
		hours[k] = hs
	}
	look := func(lat, lon float64, ts int64) (weather.Hour, bool) {
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
	}
	wp := make([]windcalc.Point, 0, len(pts))
	for _, p := range pts {
		wp = append(wp, windcalc.Point{TS: p.ts, Lat: p.lat, Lon: p.lon, Speed: p.speed})
	}
	return windcalc.Estimate(wp, look)
}

func abs64(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}
