package routeest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Drive is the length in kilometers and the driving time in minutes of the quickest road through the points, without the road's
// shape (which a plan does not need), so the answer is small.
func (c *Client) Drive(ctx context.Context, via []Point) (km, minutes float64, err error) {
	if len(via) < 2 {
		return 0, 0, fmt.Errorf("need at least two points")
	}
	base := strings.TrimRight(c.BaseURL, "/")
	if base == "" {
		base = DefaultURL
	}
	parts := make([]string, len(via))
	for i, p := range via {
		parts[i] = fmt.Sprintf("%.5f,%.5f", p[1], p[0]) // OSRM wants longitude first
	}
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", base+"/route/v1/driving/"+strings.Join(parts, ";")+"?overview=false&steps=false", nil)
	if err != nil {
		return 0, 0, err
	}
	req.Header.Set("User-Agent", "CarTrak (self-hosted trip log)")
	hc := c.HTTP
	if hc == nil {
		hc = http.DefaultClient
	}
	resp, err := hc.Do(req)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("the routing service answered HTTP %d", resp.StatusCode)
	}
	var a struct {
		Code   string `json:"code"`
		Routes []struct {
			Distance float64 `json:"distance"` // meters
			Duration float64 `json:"duration"` // seconds
		} `json:"routes"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&a); err != nil {
		return 0, 0, err
	}
	if a.Code != "Ok" || len(a.Routes) == 0 || a.Routes[0].Distance <= 0 {
		return 0, 0, fmt.Errorf("the routing service found no route (%s)", a.Code)
	}
	return a.Routes[0].Distance / 1000, a.Routes[0].Duration / 60, nil
}
