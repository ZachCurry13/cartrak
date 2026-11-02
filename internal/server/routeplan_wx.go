package server

import (
	"context"
	"math"
	"time"

	"github.com/ZachCurry13/cartrak/internal/insights"
	"github.com/ZachCurry13/cartrak/internal/weather"
	"github.com/ZachCurry13/cartrak/internal/windcalc"
)

// stopPoint is a place on the route.
type stopPoint struct {
	Name string  `json:"name"`
	Lat  float64 `json:"lat"`
	Lon  float64 `json:"lon"`
}

// legWeather is the weather expected on one leg and what it is worth in energy.
type legWeather struct {
	Known    bool     `json:"known"`
	Label    string   `json:"label,omitempty"`
	Sun      string   `json:"sun,omitempty"`
	TempC    *float64 `json:"tempC"`
	TempBand string   `json:"tempBand,omitempty"` // the temperature group of the car's own history this leg is judged by
	HeadKph  float64  `json:"headKph"`
	GustKph  float64  `json:"gustKph"`
	RainMmH  float64  `json:"rainMmH"`
	WindPct  float64  `json:"windPct"`
	RainPct  float64  `json:"rainPct"`
	Learned  bool     `json:"learned"`
}

// forecast fetches the hours of each cell and day a route touches once, and finds the nearest hour to a place and time. Forecast
// answers are not kept: the same hour asked about tomorrow is a better forecast.
type forecast struct {
	s     *Server
	ctx   context.Context
	hours map[string][]weather.Hour
	tried map[string]bool
}

func (s *Server) newForecast(ctx context.Context) *forecast {
	return &forecast{s: s, ctx: ctx, hours: map[string][]weather.Hour{}, tried: map[string]bool{}}
}

func (f *forecast) look(lat, lon float64, ts int64) (weather.Hour, bool) {
	clat, clon, cell := weather.Cell(lat, lon)
	day := time.Unix(ts, 0).UTC().Truncate(24 * time.Hour)
	key := cell + "|" + day.Format("2006-01-02")
	if !f.tried[key] {
		f.tried[key] = true
		if hs, err := f.s.weatherClient().Day(f.ctx, clat, clon, day); err == nil {
			f.hours[key] = hs
		}
	}
	hs := f.hours[key]
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

// legSteps lays points along the straight line between two stops, a few minutes apart, so the weather and wind can be met hour by
// hour on a long leg.
func legSteps(a, b stopPoint, depart int64, minutes, kph float64) []weather.Pt {
	n := int(math.Ceil(minutes/4)) + 1
	if n < 2 {
		n = 2
	}
	pts := make([]weather.Pt, n)
	for i := range pts {
		f := float64(i) / float64(n-1)
		pts[i] = weather.Pt{TS: depart + int64(f*minutes*60), Lat: a.Lat + (b.Lat-a.Lat)*f, Lon: a.Lon + (b.Lon-a.Lon)*f, Speed: kph}
	}
	return pts
}

// weatherForLeg is what the forecast says about driving from a to b leaving at depart, as an energy effect on top of how the car
// usually drives in that temperature.
func (s *Server) weatherForLeg(fc *forecast, a, b stopPoint, depart int64, km, minutes float64) (legWeather, float64) {
	var lw legWeather
	if fc == nil || minutes <= 0 {
		return lw, 0
	}
	kph := km / minutes * 60
	pts := legSteps(a, b, depart, minutes, kph)
	c, ok := weather.Summarize(pts, fc.look)
	if !ok {
		return lw, 0
	}
	lw.Known, lw.Label, lw.Sun, lw.RainMmH = true, c.Label, c.Sun, c.RainMmH
	var sum float64
	n := 0
	for _, p := range pts {
		if h, ok := fc.look(p.Lat, p.Lon, p.TS); ok && h.TempC != nil {
			sum += *h.TempC
			n++
		}
	}
	if n > 0 {
		t := sum / float64(n)
		lw.TempC = &t
		lw.TempBand = insights.BandName(t)
	}
	wp := make([]windcalc.Point, len(pts))
	for i, p := range pts {
		wp[i] = windcalc.Point{TS: p.TS, Lat: p.Lat, Lon: p.Lon, Speed: p.Speed}
	}
	if w, ok := windcalc.Estimate(wp, windcalc.Lookup(fc.look)); ok {
		lw.HeadKph, lw.GustKph, lw.WindPct = w.HeadKph, w.GustKph, w.EffectPct
	}
	scale := 1.0
	if l, learned := s.learnedRain(); learned && !c.Snowy {
		scale, lw.Learned = l.Scale, true
	}
	lw.RainPct = c.Effect(scale)
	return lw, (1 + lw.WindPct/100) * (1 + lw.RainPct/100)
}
