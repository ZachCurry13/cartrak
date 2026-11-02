package server

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/ZachCurry13/cartrak/internal/insights"
	"github.com/ZachCurry13/cartrak/internal/trips"
	"github.com/ZachCurry13/cartrak/internal/tripwhy"
	"github.com/ZachCurry13/cartrak/internal/weather"
)

const (
	learnTTL        = 6 * time.Hour
	backfillDays    = 120 // how far back the background lookup fills in weather for past trips
	backfillPerRun  = 12  // days of weather asked for in one run, so the free service is never hammered
	backfillEvery   = 30 * time.Minute
	backfillSpacing = 1200 * time.Millisecond
)

// rainLearn is what the owner's own wet and dry trips say about rain, kept for a few hours.
type rainLearn struct {
	mu  sync.Mutex
	at  time.Time
	val weather.Learned
	ok  bool
}

// learnedRain compares the efficiency of past trips that met rain with the dry ones, using only weather that is already kept.
func (s *Server) learnedRain() (weather.Learned, bool) {
	c := &s.rainLearn
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.at.IsZero() && time.Since(c.at) < learnTTL {
		return c.val, c.ok
	}
	c.at = time.Now()
	c.val, c.ok = weather.Learned{}, false
	from := time.Now().AddDate(0, 0, -366).Unix()
	ts, err := s.S.EnergyTrips(from)
	if err != nil {
		return c.val, false
	}
	temps, _ := s.S.TripTemps(from)
	base, capacity := s.whyBaseline(), s.capacityKwh()
	var samples []weather.RainSample
	for _, t := range ts {
		if t.Open || t.EndTS == nil || t.StartLat == nil || t.StartLon == nil || t.DistanceKm < 3 {
			continue
		}
		e, ok := trips.EnergyUsed(t, capacity)
		hours := float64(*t.EndTS-t.StartTS) / 3600
		if !ok || e.Kwh <= 0 || hours <= 0 {
			continue
		}
		group := ""
		if tc, has := temps[t.ID]; has {
			group = insights.BandName(tc)
		}
		exp := base.ExpectedKmPerKwh(t.DistanceKm/hours, group)
		mm, known := s.rainDuring(*t.StartLat, *t.StartLon, t.StartTS, *t.EndTS)
		if exp <= 0 || !known {
			continue
		}
		samples = append(samples, weather.RainSample{RainMmH: mm, Ratio: (t.DistanceKm / e.Kwh) / exp})
	}
	c.val, c.ok = weather.LearnRain(samples)
	return c.val, c.ok
}

// rainDuring is the average rain, in millimeters an hour, over a trip's hours from the weather already kept for where it started,
// or false when that is not kept, the trip crosses midnight UTC, or it snowed.
func (s *Server) rainDuring(lat, lon float64, start, end int64) (float64, bool) {
	day := time.Unix(start, 0).UTC().Truncate(24 * time.Hour)
	if end >= day.Add(24*time.Hour).Unix() {
		return 0, false
	}
	_, _, cell := weather.Cell(lat, lon)
	hs, ok := s.weatherCached(cell, day)
	if !ok {
		return 0, false
	}
	var sum float64
	n := 0
	for _, h := range hs {
		if h.TS+3600 <= start || h.TS > end {
			continue
		}
		if h.SnowCm >= 0.1 {
			return 0, false
		}
		sum += h.RainMm
		n++
	}
	if n == 0 {
		return 0, false
	}
	return sum / float64(n), true
}

// tripWeather is the rain or snow met on a trip and what it cost, or nil when the weather lookup is off or has no answer.
func (s *Server) tripWeather(pts []windPoint, look hourLook) *tripwhy.Weather {
	if look == nil {
		return nil
	}
	wp := make([]weather.Pt, 0, len(pts))
	for _, p := range pts {
		wp = append(wp, weather.Pt{TS: p.ts, Lat: p.lat, Lon: p.lon, Speed: p.speed})
	}
	c, ok := weather.Summarize(wp, look)
	if !ok {
		return nil
	}
	w := &tripwhy.Weather{Label: c.Label, Dry: c.Dry, Snowy: c.Snowy, RainMmH: c.RainMmH, SnowCmH: c.SnowCmH, WetShare: c.WetShare}
	scale := 1.0
	if l, learned := s.learnedRain(); learned && !c.Snowy { // what the owner's own trips measured replaces the fixed estimate for rain
		scale, w.Learned, w.WetTrips = l.Scale, true, l.WetTrips
	}
	w.EffectPct = c.Effect(scale)
	return w
}

// WeatherBackfill fills in the weather for recent past trips in the background, a few days of it at a time, so that what rain
// costs can be measured from the owner's own trips and not only guessed. It does nothing unless the weather lookup is on.
func (s *Server) WeatherBackfill(ctx context.Context) {
	select {
	case <-time.After(2 * time.Minute):
	case <-ctx.Done():
		return
	}
	for {
		s.backfillWeather(ctx)
		select {
		case <-time.After(backfillEvery):
		case <-ctx.Done():
			return
		}
	}
}

func (s *Server) backfillWeather(ctx context.Context) {
	if s.Cfg == nil || !s.Cfg.WeatherLookup() {
		return
	}
	ts, err := s.S.EnergyTrips(time.Now().AddDate(0, 0, -backfillDays).Unix())
	if err != nil {
		return
	}
	sort.Slice(ts, func(i, j int) bool { return ts[i].StartTS > ts[j].StartTS }) // newest first
	client := s.weatherClient()
	seen := map[string]bool{}
	fetched := 0
	for _, t := range ts {
		if ctx.Err() != nil || fetched >= backfillPerRun {
			break
		}
		if t.StartLat == nil || t.StartLon == nil {
			continue
		}
		day := time.Unix(t.StartTS, 0).UTC().Truncate(24 * time.Hour)
		_, _, cell := weather.Cell(*t.StartLat, *t.StartLon)
		key := cell + "|" + day.Format("2006-01-02")
		if seen[key] {
			continue
		}
		seen[key] = true
		if _, kept := s.weatherCached(cell, day); kept {
			continue
		}
		fetched++
		s.weatherDay(ctx, client, *t.StartLat, *t.StartLon, day, true)
		select {
		case <-time.After(backfillSpacing):
		case <-ctx.Done():
			return
		}
	}
	if fetched > 0 {
		s.rainLearn.mu.Lock()
		s.rainLearn.at = time.Time{} // new weather is kept: measure again at the next trip opened
		s.rainLearn.mu.Unlock()
	}
}
