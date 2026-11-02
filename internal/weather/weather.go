// Package weather looks up the hourly wind, rain and snow for a place and a day, for the optional weather on a trip. It only
// asks for wind speed, direction and gusts, rain, snowfall and a weather code, and only for a map cell about fifty kilometers
// wide, never an exact position.
package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Hour is the weather in one UTC hour.
type Hour struct {
	TS      int64   `json:"ts"`
	SpeedKm float64 `json:"speedKph"` // 10 m wind speed
	DirDeg  float64 `json:"dirDeg"`   // the direction the wind comes FROM
	GustKm  float64 `json:"gustKph"`
	// What was asked for: V is 2 on every hour of an answer that asked about rain and snow, so a day saved before then (V 0) is
	// asked for again, while a service that sends no rain figures is not asked again and again.
	V      int      `json:"v,omitempty"`
	Wx     bool     `json:"wx,omitempty"`
	RainMm float64  `json:"rainMm,omitempty"` // precipitation in the hour, millimeters (rain and melted snow)
	SnowCm float64  `json:"snowCm,omitempty"` // snowfall in the hour, centimeters
	Sun    float64  `json:"sun,omitempty"`    // sunshine on the ground, watts per square meter (0 at night)
	TempC  *float64 `json:"tempC,omitempty"`  // air temperature, for a forecast
	Code   int      `json:"code,omitempty"`   // WMO weather code: 61 rain, 71 snow, 95 thunderstorm, and so on
}

// Cell is a map cell about fifty kilometers wide (half a degree): the only form of a position that is ever sent.
func Cell(lat, lon float64) (clat, clon float64, key string) {
	clat, clon = math.Round(lat*2)/2, math.Round(lon*2)/2
	return clat, clon, fmt.Sprintf("%.1f,%.1f", clat, clon)
}

// Client asks Open-Meteo, or any service with the same answer.
type Client struct {
	HTTP        *http.Client
	ForecastURL string // recent days
	ArchiveURL  string // older days
}

// Default addresses (Open-Meteo is free and needs no account).
const (
	DefaultForecast = "https://api.open-meteo.com/v1/forecast"
	DefaultArchive  = "https://archive-api.open-meteo.com/v1/archive"
)

type answer struct {
	Hourly struct {
		Time []string  `json:"time"`
		Wind []float64 `json:"wind_speed_10m"`
		Dir  []float64 `json:"wind_direction_10m"`
		Gust []float64 `json:"wind_gusts_10m"`
		Rain []float64 `json:"precipitation"`
		Snow []float64 `json:"snowfall"`
		Code []float64 `json:"weather_code"`
		Sun  []float64 `json:"shortwave_radiation"`
		Temp []float64 `json:"temperature_2m"`
	} `json:"hourly"`
}

// Day fetches the 24 hours of one UTC day for a cell. Days in the last two weeks come from the forecast service, which
// has recent days, older ones from the archive.
func (c *Client) Day(ctx context.Context, lat, lon float64, day time.Time) ([]Hour, error) {
	base := c.ArchiveURL
	if base == "" {
		base = DefaultArchive
	}
	if time.Since(day) < 14*24*time.Hour {
		base = c.ForecastURL
		if base == "" {
			base = DefaultForecast
		}
	}
	d := day.UTC().Format("2006-01-02")
	q := url.Values{"latitude": {fmt.Sprintf("%.1f", lat)}, "longitude": {fmt.Sprintf("%.1f", lon)}, "hourly": {"wind_speed_10m,wind_direction_10m,wind_gusts_10m,precipitation,snowfall,weather_code,shortwave_radiation,temperature_2m"},
		"wind_speed_unit": {"kmh"}, "timezone": {"UTC"}, "start_date": {d}, "end_date": {d}}
	req, err := http.NewRequestWithContext(ctx, "GET", base+"?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	hc := c.HTTP
	if hc == nil {
		hc = &http.Client{Timeout: 10 * time.Second}
	}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("weather service answered %d", resp.StatusCode)
	}
	hs, err := Parse(body)
	for i := range hs {
		hs[i].V = 2
	}
	return hs, err
}

// Parse reads the service's answer into hours.
func Parse(body []byte) ([]Hour, error) {
	var a answer
	if err := json.Unmarshal(body, &a); err != nil {
		return nil, err
	}
	var out []Hour
	for i, ts := range a.Hourly.Time {
		t, err := time.Parse("2006-01-02T15:04", strings.TrimSuffix(ts, "Z"))
		if err != nil || i >= len(a.Hourly.Wind) || i >= len(a.Hourly.Dir) {
			continue
		}
		h := Hour{TS: t.UTC().Unix(), SpeedKm: a.Hourly.Wind[i], DirDeg: a.Hourly.Dir[i]}
		if i < len(a.Hourly.Gust) {
			h.GustKm = a.Hourly.Gust[i]
		}
		if i < len(a.Hourly.Rain) && i < len(a.Hourly.Snow) { // both present means the service answered about rain and snow
			h.Wx, h.RainMm, h.SnowCm = true, a.Hourly.Rain[i], a.Hourly.Snow[i]
			if i < len(a.Hourly.Code) {
				h.Code = int(a.Hourly.Code[i])
			}
			if i < len(a.Hourly.Sun) {
				h.Sun = a.Hourly.Sun[i]
			}
			if i < len(a.Hourly.Temp) {
				t := a.Hourly.Temp[i]
				h.TempC = &t
			}
		}
		out = append(out, h)
	}
	return out, nil
}
