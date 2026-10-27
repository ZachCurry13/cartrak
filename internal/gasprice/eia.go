package gasprice

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

// DefaultURL is the EIA open data service.
const DefaultURL = "https://api.eia.gov"

// Point is the average price (dollars per gallon) for the week starting on Date.
type Point struct {
	Date  string  `json:"d"` // 2026-10-19
	Price float64 `json:"v"`
}

// Fetch asks the EIA for the weekly regular-gasoline price of an area since the date. The key travels in the address, as the
// EIA requires, so errors are rewritten to never contain it.
func Fetch(ctx context.Context, hc *http.Client, base, key string, a Area, since time.Time) ([]Point, error) {
	base = strings.TrimRight(base, "/")
	if base == "" {
		base = DefaultURL
	}
	if hc == nil {
		hc = &http.Client{Timeout: 20 * time.Second}
	}
	q := url.Values{"api_key": {key}, "frequency": {"weekly"}, "data[0]": {"value"}, "facets[product][]": {"EPMR"},
		"facets[duoarea][]": {a.ID}, "start": {since.Format("2006-01-02")}, "sort[0][column]": {"period"}, "sort[0][direction]": {"asc"}, "length": {"5000"}}
	req, err := http.NewRequestWithContext(ctx, "GET", base+"/v2/petroleum/pri/gnd/data/?"+q.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("the gas price request could not be made")
	}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("the gas price service could not be reached")
	}
	defer resp.Body.Close()
	if resp.StatusCode == 403 || resp.StatusCode == 401 {
		return nil, fmt.Errorf("the gas price service refused the EIA key")
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("the gas price service answered HTTP %d", resp.StatusCode)
	}
	var out struct {
		Response struct {
			Data []struct {
				Period string `json:"period"`
				Value  any    `json:"value"`
			} `json:"data"`
		} `json:"response"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&out); err != nil {
		return nil, fmt.Errorf("the gas price service sent something unreadable")
	}
	var pts []Point
	for _, d := range out.Response.Data {
		var v float64
		switch x := d.Value.(type) {
		case float64:
			v = x
		case string:
			v, _ = strconv.ParseFloat(x, 64)
		}
		if v > 0 && v < 30 && d.Period != "" {
			pts = append(pts, Point{d.Period, v})
		}
	}
	return pts, nil
}
