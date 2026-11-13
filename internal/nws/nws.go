// Package nws asks the US National Weather Service for the warnings in force at a place: winter storms, ice, floods, high wind and
// the like. It is free and needs no account. Only a position rounded to a tenth of a degree (about 10 km) is sent.
package nws

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"
)

// DefaultURL is the National Weather Service API.
const DefaultURL = "https://api.weather.gov"

// Alert is one warning, watch or advisory.
type Alert struct {
	ID       string `json:"id"`
	Event    string `json:"event"`    // "Winter Storm Warning"
	Severity string `json:"severity"` // Extreme, Severe, Moderate, Minor or Unknown
	Headline string `json:"headline"`
	Area     string `json:"area"`
	Onset    int64  `json:"onset,omitempty"`
	Ends     int64  `json:"ends,omitempty"`
	Drive    bool   `json:"drive"` // about the roads, and serious enough to be told about (see ForDriving)
}

// Round is the position actually sent: a tenth of a degree.
func Round(x float64) float64 { return math.Round(x*10) / 10 }

// Active lists the warnings in force at a place, the most severe first.
func Active(ctx context.Context, hc *http.Client, base string, lat, lon float64) ([]Alert, error) {
	base = strings.TrimRight(base, "/")
	if base == "" {
		base = DefaultURL
	}
	if hc == nil {
		hc = &http.Client{Timeout: 15 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/alerts/active?point=%.1f,%.1f", base, Round(lat), Round(lon)), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "CarTrak (self-hosted car tracker)") // the service asks every caller to name itself
	req.Header.Set("Accept", "application/geo+json")
	resp, err := hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("the weather service could not be reached")
	}
	defer resp.Body.Close()
	if resp.StatusCode == 400 || resp.StatusCode == 404 {
		return nil, fmt.Errorf("the weather service only covers the United States")
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("the weather service answered HTTP %d", resp.StatusCode)
	}
	var out struct {
		Features []struct {
			Properties struct {
				ID, Event, Severity, Headline, AreaDesc, Status, MessageType string
				Onset, Effective, Ends, Expires                              string
			} `json:"properties"`
		} `json:"features"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&out); err != nil {
		return nil, fmt.Errorf("the weather service sent something unreadable")
	}
	list := []Alert{}
	for _, f := range out.Features {
		p := f.Properties
		if !strings.EqualFold(p.Status, "Actual") || strings.EqualFold(p.MessageType, "Cancel") {
			continue // tests, exercises and cancellations
		}
		a := Alert{ID: p.ID, Event: p.Event, Severity: p.Severity, Headline: p.Headline, Area: p.AreaDesc, Onset: unix(p.Onset, p.Effective), Ends: unix(p.Ends, p.Expires)}
		a.Drive = ForDriving(a)
		list = append(list, a)
	}
	sort.SliceStable(list, func(i, j int) bool { return rank(list[i].Severity) > rank(list[j].Severity) })
	return list, nil
}

func unix(vals ...string) int64 {
	for _, v := range vals {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			return t.Unix()
		}
	}
	return 0
}

func rank(s string) int {
	return map[string]int{"Extreme": 4, "Severe": 3, "Moderate": 2, "Minor": 1}[s]
}

// driving are the events that change a drive: worth an alert on the phone. Others (heat, air quality, rip currents) are only shown.
var driving = []string{"winter", "ice", "snow", "blizzard", "freez", "sleet", "flood", "fog", "wind", "tornado", "thunderstorm", "dust", "hurricane", "tropical"}

// ForDriving says whether a warning is about the roads, and serious enough to be told about (an advisory or more).
func ForDriving(a Alert) bool {
	e := strings.ToLower(a.Event)
	if strings.Contains(e, "statement") || strings.Contains(e, "outlook") || rank(a.Severity) < 2 {
		return false
	}
	for _, w := range driving {
		if strings.Contains(e, w) {
			return true
		}
	}
	return false
}

// Ping asks once for a point in the middle of the country, for the Integrations test.
func Ping(ctx context.Context, hc *http.Client, base string) error {
	_, err := Active(ctx, hc, base, 39.8, -98.6)
	return err
}
