// Package weather looks up the hourly wind for a place and a day, for the optional wind estimate on a trip. It only
// asks for wind speed, direction and gusts, and only for a map cell about fifty kilometers wide, never an exact position.
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

// Hour is the wind in one UTC hour.
type Hour struct {
	TS      int64   `json:"ts"`
	SpeedKm float64 `json:"speedKph"` // 10 m wind speed
	DirDeg  float64 `json:"dirDeg"`   // the direction the wind comes FROM
	GustKm  float64 `json:"gustKph"`
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
	q := url.Values{"latitude": {fmt.Sprintf("%.1f", lat)}, "longitude": {fmt.Sprintf("%.1f", lon)}, "hourly": {"wind_speed_10m,wind_direction_10m,wind_gusts_10m"},
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
	return Parse(body)
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
		out = append(out, h)
	}
	return out, nil
}
