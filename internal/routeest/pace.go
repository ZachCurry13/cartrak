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

// Pace is how a drive divides between road driven at highway speed and the rest, as the routing service sees it.
type Pace struct {
	Km, Minutes           float64 // the whole drive, as the routing service times it
	HighwayKm, HighwayMin float64 // the part on roads where the routing service drives 70 km/h or faster, and the time it takes there
}

// highwayMps is the speed from which a stretch of road counts as highway (70 km/h).
const highwayMps = 70 / 3.6

// DrivePace is Drive with the speed of each stretch of road, so the part on highways can be driven faster than the routing service's
// speed. The routing service gives its own speed for each stretch (from the map's speed limits where they are tagged, with its own
// allowances), not the posted limit itself.
func (c *Client) DrivePace(ctx context.Context, via []Point) (Pace, error) {
	if len(via) < 2 {
		return Pace{}, fmt.Errorf("need at least two points")
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
	req, err := http.NewRequestWithContext(ctx, "GET", base+"/route/v1/driving/"+strings.Join(parts, ";")+"?overview=false&steps=false&annotations=speed,distance", nil)
	if err != nil {
		return Pace{}, err
	}
	req.Header.Set("User-Agent", "CarTrak (self-hosted trip log)")
	hc := c.HTTP
	if hc == nil {
		hc = http.DefaultClient
	}
	resp, err := hc.Do(req)
	if err != nil {
		return Pace{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Pace{}, fmt.Errorf("the routing service answered HTTP %d", resp.StatusCode)
	}
	var a struct {
		Code   string `json:"code"`
		Routes []struct {
			Distance float64 `json:"distance"`
			Duration float64 `json:"duration"`
			Legs     []struct {
				Annotation struct {
					Distance []float64 `json:"distance"`
					Speed    []float64 `json:"speed"`
				} `json:"annotation"`
			} `json:"legs"`
		} `json:"routes"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 16<<20)).Decode(&a); err != nil {
		return Pace{}, err
	}
	if a.Code != "Ok" || len(a.Routes) == 0 || a.Routes[0].Distance <= 0 {
		return Pace{}, fmt.Errorf("the routing service found no route (%s)", a.Code)
	}
	r := a.Routes[0]
	out := Pace{Km: r.Distance / 1000, Minutes: r.Duration / 60}
	for _, leg := range r.Legs {
		for i, d := range leg.Annotation.Distance {
			if i >= len(leg.Annotation.Speed) || leg.Annotation.Speed[i] < highwayMps {
				continue
			}
			out.HighwayKm += d / 1000
			out.HighwayMin += d / leg.Annotation.Speed[i] / 60
		}
	}
	return out, nil
}
