// Package routeest estimates the road a trip took when only a few positions were recorded, by asking a routing service
// (OSRM, or anything with the same answer) for the road between them. It is an estimate of the likely road, not a
// recording of where the car went, and it is only used for trips that have no phone GPS track.
package routeest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"
)

// DefaultURL is the public OSRM demo server, which is free but meant for light use. Point CarTrak at your own to avoid it.
const DefaultURL = "https://router.project-osrm.org"

// Point is a latitude and longitude.
type Point [2]float64

// Client asks an OSRM-compatible server for a driving route.
type Client struct {
	HTTP    *http.Client
	BaseURL string
}

type answer struct {
	Code   string `json:"code"`
	Routes []struct {
		Distance float64 `json:"distance"` // metres
		Geometry struct {
			Coordinates [][2]float64 `json:"coordinates"` // lon, lat
		} `json:"geometry"`
	} `json:"routes"`
}

// Route is the driving route through the points in order, and its length in kilometres.
func (c *Client) Route(ctx context.Context, via []Point) ([]Point, float64, error) {
	if len(via) < 2 {
		return nil, 0, fmt.Errorf("need at least two points")
	}
	base := strings.TrimRight(c.BaseURL, "/")
	if base == "" {
		base = DefaultURL
	}
	parts := make([]string, len(via))
	for i, p := range via {
		parts[i] = fmt.Sprintf("%.5f,%.5f", p[1], p[0]) // OSRM wants longitude first
	}
	url := base + "/route/v1/driving/" + strings.Join(parts, ";") + "?overview=full&geometries=geojson&steps=false"
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", "CarTrak (self-hosted trip log)")
	hc := c.HTTP
	if hc == nil {
		hc = http.DefaultClient
	}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("the routing service answered HTTP %d", resp.StatusCode)
	}
	var a answer
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(&a); err != nil {
		return nil, 0, err
	}
	if a.Code != "Ok" || len(a.Routes) == 0 || len(a.Routes[0].Geometry.Coordinates) < 2 {
		return nil, 0, fmt.Errorf("the routing service found no route (%s)", a.Code)
	}
	r := a.Routes[0]
	out := make([]Point, len(r.Geometry.Coordinates))
	for i, c := range r.Geometry.Coordinates {
		out[i] = Point{c[1], c[0]}
	}
	return out, r.Distance / 1000, nil
}

// Via picks the points to route through: the first, the last and up to max-2 spread between, from the positions the car
// did record. More points keep the estimate on the road the car really used.
func Via(pts []Point, max int) []Point {
	if len(pts) <= max || max < 2 {
		return pts
	}
	out := make([]Point, 0, max)
	step := float64(len(pts)-1) / float64(max-1)
	for i := 0; i < max; i++ {
		out = append(out, pts[int(float64(i)*step+0.5)])
	}
	return out
}

// Plausible says whether a route's length is believable for the trip it is meant to describe, judged against the
// distance the odometer recorded, or against the straight line from start to end when the odometer is plainly short
// (it sometimes updates late, and a road is always longer than the straight line). A route far longer or shorter would
// mislead, so it is not drawn.
func Plausible(routeKm, tripKm, straightKm float64) bool {
	if tripKm < 0.5 || routeKm <= 0 {
		return false
	}
	r := routeKm / math.Max(tripKm, straightKm*1.3)
	return r >= 0.6 && r <= 1.6 && !math.IsNaN(r)
}

// Thin keeps at most max points, evenly spread, always including the first and last.
func Thin(pts []Point, max int) []Point { return Via(pts, max) }
