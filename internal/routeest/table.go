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

// Table is the driving time in minutes and the distance in kilometers from every point to every other, in one request, for
// working out the best order to visit places in.
func (c *Client) Table(ctx context.Context, via []Point) (minutes, km [][]float64, err error) {
	if len(via) < 2 || len(via) > 25 {
		return nil, nil, fmt.Errorf("need between two and 25 points")
	}
	base := strings.TrimRight(c.BaseURL, "/")
	if base == "" {
		base = DefaultURL
	}
	parts := make([]string, len(via))
	for i, p := range via {
		parts[i] = fmt.Sprintf("%.5f,%.5f", p[1], p[0]) // OSRM wants longitude first
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", base+"/table/v1/driving/"+strings.Join(parts, ";")+"?annotations=duration,distance", nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("User-Agent", "CarTrak (self-hosted trip log)")
	hc := c.HTTP
	if hc == nil {
		hc = http.DefaultClient
	}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("the routing service answered HTTP %d", resp.StatusCode)
	}
	var a struct {
		Code      string       `json:"code"`
		Durations [][]*float64 `json:"durations"`
		Distances [][]*float64 `json:"distances"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&a); err != nil {
		return nil, nil, err
	}
	n := len(via)
	if a.Code != "Ok" || len(a.Durations) != n || len(a.Distances) != n {
		return nil, nil, fmt.Errorf("the routing service gave no table (%s)", a.Code)
	}
	minutes, km = make([][]float64, n), make([][]float64, n)
	for i := 0; i < n; i++ {
		if len(a.Durations[i]) != n || len(a.Distances[i]) != n {
			return nil, nil, fmt.Errorf("the routing service gave a short table")
		}
		minutes[i], km[i] = make([]float64, n), make([]float64, n)
		for j := 0; j < n; j++ {
			if a.Durations[i][j] == nil || a.Distances[i][j] == nil {
				return nil, nil, fmt.Errorf("the routing service could not connect two of the places")
			}
			minutes[i][j], km[i][j] = *a.Durations[i][j]/60, *a.Distances[i][j]/1000
		}
	}
	return minutes, km, nil
}
