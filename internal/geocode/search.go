package geocode

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Match is one place an address could mean.
type Match struct {
	Label string  `json:"label"`
	Lat   float64 `json:"lat"`
	Lon   float64 `json:"lon"`
}

// Search turns a typed address into up to five places. What was typed goes to the service and nowhere else: unlike the city
// lookups it is not remembered, because an address is something a person typed, not a rounded position.
func (c *Client) Search(ctx context.Context, query string) ([]Match, error) {
	query = strings.TrimSpace(query)
	if len(query) < 3 || len(query) > 200 {
		return nil, fmt.Errorf("type at least a street and a town")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.pace(ctx); err != nil {
		return nil, err
	}
	q := url.Values{"format": {"jsonv2"}, "limit": {"5"}, "accept-language": {"en"}, "q": {query}}
	req, err := http.NewRequestWithContext(ctx, "GET", c.base()+"/search?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.UserAgent)
	hc := c.HTTP
	if hc == nil {
		hc = &http.Client{Timeout: 15 * time.Second}
	}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("the address lookup answered HTTP %d", resp.StatusCode)
	}
	var raw []struct {
		Lat, Lon    string
		DisplayName string `json:"display_name"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&raw); err != nil {
		return nil, err
	}
	out := []Match{}
	for _, r := range raw {
		lat, e1 := strconv.ParseFloat(r.Lat, 64)
		lon, e2 := strconv.ParseFloat(r.Lon, 64)
		if e1 != nil || e2 != nil || lat < -90 || lat > 90 || lon < -180 || lon > 180 {
			continue
		}
		out = append(out, Match{Label: r.DisplayName, Lat: lat, Lon: lon})
	}
	return out, nil
}
