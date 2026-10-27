// Package geocode finds the city a place is in, for the mileage report. It asks a Nominatim
// reverse-geocoding service (OpenStreetMap's by default) and remembers every answer, so each
// place is looked up once. Locations are rounded to about 100 m before they leave the server.
package geocode

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// DefaultURL is OpenStreetMap's public Nominatim service. Its usage policy asks for at most one
// request a second, an identifying User-Agent and caching of results; all three are done here.
const DefaultURL = "https://nominatim.openstreetmap.org"

// Cache stores answers between runs.
type Cache interface {
	GetPlace(key string) (city string, found bool)
	SetPlace(key, city string) error
}

type Client struct {
	BaseURL   string // empty means DefaultURL
	UserAgent string
	Cache     Cache
	HTTP      *http.Client  // optional
	MinGap    time.Duration // between requests; defaults to 1.1 s

	mu   sync.Mutex
	last time.Time
}

// Key is the rounded place a lookup is made and cached for.
func Key(lat, lon float64) string { return fmt.Sprintf("%.3f,%.3f", lat, lon) }

// City returns the city (or town, village or municipality) at the place, "" if the service knows
// none. Answers come from the cache when possible.
func (c *Client) City(ctx context.Context, lat, lon float64) (string, error) {
	key := Key(lat, lon)
	if c.Cache != nil {
		if city, ok := c.Cache.GetPlace(key); ok {
			return city, nil
		}
	}
	city, state, err := c.fetch(ctx, lat, lon)
	if err != nil {
		return "", err
	}
	if c.Cache != nil {
		_ = c.Cache.SetPlace(key, city)
		_ = c.Cache.SetPlace(stateKey(key), state)
	}
	return city, nil
}

func stateKey(key string) string { return "st:" + key }

// State is the two-letter code of the US state at the place ("MN"), "" outside the US or when the service knows none.
// It is remembered next to the city, so a place already looked up for its city costs no second request.
func (c *Client) State(ctx context.Context, lat, lon float64) (string, error) {
	key := Key(lat, lon)
	if c.Cache != nil {
		if st, ok := c.Cache.GetPlace(stateKey(key)); ok {
			return st, nil
		}
	}
	city, state, err := c.fetch(ctx, lat, lon)
	if err != nil {
		return "", err
	}
	if c.Cache != nil {
		_ = c.Cache.SetPlace(key, city)
		_ = c.Cache.SetPlace(stateKey(key), state)
	}
	return state, nil
}

// CachedState returns a remembered state without asking the service.
func (c *Client) CachedState(lat, lon float64) (string, bool) {
	if c.Cache == nil {
		return "", false
	}
	return c.Cache.GetPlace(stateKey(Key(lat, lon)))
}

func (c *Client) fetch(ctx context.Context, lat, lon float64) (string, string, error) {
	c.mu.Lock() // one lookup at a time, spaced out
	defer c.mu.Unlock()
	gap := c.MinGap
	if gap == 0 {
		gap = 1100 * time.Millisecond
	}
	if wait := gap - time.Since(c.last); wait > 0 {
		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return "", "", ctx.Err()
		}
	}
	c.last = time.Now()

	base := strings.TrimRight(c.BaseURL, "/")
	if base == "" {
		base = DefaultURL
	}
	q := url.Values{"format": {"jsonv2"}, "zoom": {"10"}, "addressdetails": {"1"}, "accept-language": {"en"},
		"lat": {fmt.Sprintf("%.3f", lat)}, "lon": {fmt.Sprintf("%.3f", lon)}}
	req, err := http.NewRequestWithContext(ctx, "GET", base+"/reverse?"+q.Encode(), nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("User-Agent", c.UserAgent)
	hc := c.HTTP
	if hc == nil {
		hc = &http.Client{Timeout: 15 * time.Second}
	}
	resp, err := hc.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", "", fmt.Errorf("the place lookup answered HTTP %d", resp.StatusCode)
	}
	var out struct {
		Address map[string]string `json:"address"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&out); err != nil {
		return "", "", err
	}
	return pickCity(out.Address), pickState(out.Address), nil
}

// pickState reads the US state code from the ISO 3166-2 field ("US-MN"); other countries give "".
func pickState(a map[string]string) string {
	if v := strings.TrimSpace(a["ISO3166-2-lvl4"]); strings.HasPrefix(v, "US-") && len(v) == 5 {
		return v[3:]
	}
	return ""
}

// pickCity prefers the most specific named settlement. Nominatim calls it city, town, village or
// municipality depending on its size; county is the last resort.
func pickCity(a map[string]string) string {
	for _, k := range []string{"city", "town", "village", "municipality", "hamlet", "suburb", "county"} {
		if v := strings.TrimSpace(a[k]); v != "" {
			return v
		}
	}
	return ""
}

// Cached returns a remembered answer without asking the service.
func (c *Client) Cached(lat, lon float64) (string, bool) {
	if c.Cache == nil {
		return "", false
	}
	return c.Cache.GetPlace(Key(lat, lon))
}

// Lookup is City under the name the mileage report uses.
func (c *Client) Lookup(ctx context.Context, lat, lon float64) (string, error) {
	return c.City(ctx, lat, lon)
}

// SetBaseURL changes the service address (settings can change while the server runs).
func (c *Client) SetBaseURL(u string) {
	c.mu.Lock()
	c.BaseURL = u
	c.mu.Unlock()
}
