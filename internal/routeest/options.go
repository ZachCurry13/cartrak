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

// Option is one road for a whole trip: through the places the car reported, in order.
type Option struct {
	Kind   string  `json:"kind"` // "closest" to the odometer, "quickest" or "shortest"
	Label  string  `json:"label"`
	Km     float64 `json:"km"`
	Points []Point `json:"-"`
}

type alt struct {
	pts []Point
	km  float64
}

// alternatives are the roads the service offers between two places: the quickest first, then up to two others.
func (c *Client) alternatives(ctx context.Context, a, b Point) ([]alt, error) {
	base := strings.TrimRight(c.BaseURL, "/")
	if base == "" {
		base = DefaultURL
	}
	url := fmt.Sprintf("%s/route/v1/driving/%.5f,%.5f;%.5f,%.5f?overview=full&geometries=geojson&steps=false&alternatives=3", base, a[1], a[0], b[1], b[0]) // longitude first
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "CarTrak (self-hosted trip log)")
	hc := c.HTTP
	if hc == nil {
		hc = http.DefaultClient
	}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("the routing service answered HTTP %d", resp.StatusCode)
	}
	var out answer
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(&out); err != nil {
		return nil, err
	}
	if out.Code != "Ok" || len(out.Routes) == 0 {
		return nil, fmt.Errorf("the routing service found no route (%s)", out.Code)
	}
	var alts []alt
	for _, r := range out.Routes {
		if len(r.Geometry.Coordinates) < 2 {
			continue
		}
		pts := make([]Point, len(r.Geometry.Coordinates))
		for i, c := range r.Geometry.Coordinates {
			pts[i] = Point{c[1], c[0]}
		}
		alts = append(alts, alt{pts: pts, km: r.Distance / 1000})
	}
	if len(alts) == 0 {
		return nil, fmt.Errorf("the routing service found no route (%s)", out.Code)
	}
	return alts, nil
}

// Options works out the roads for a trip through its recorded places. Each stretch between two places is asked for with its
// alternatives, so the whole can be put together in different ways: the quickest, the shortest, and (when the odometer length is
// known) the one whose total is closest to it. A trip the odometer says was 20 km is then drawn on the roads that add up to
// about 20 km, not on the quickest ones that may add up to 26.
func (c *Client) Options(ctx context.Context, via []Point, targetKm float64) ([]Option, error) {
	if len(via) < 2 {
		return nil, fmt.Errorf("need at least two points")
	}
	segs := make([][]alt, 0, len(via)-1)
	for i := 0; i+1 < len(via); i++ {
		if i > 0 {
			time.Sleep(150 * time.Millisecond) // the free routing service asks for a gentle pace
		}
		a, err := c.alternatives(ctx, via[i], via[i+1])
		if err != nil {
			return nil, err
		}
		segs = append(segs, a)
	}
	pick := func(choose func(seg int, alts []alt) int) Option {
		var o Option
		for i, alts := range segs {
			a := alts[choose(i, alts)]
			o.Km += a.km
			if len(o.Points) > 0 {
				o.Points = append(o.Points, a.pts[1:]...) // the first place of this stretch is the last of the one before
			} else {
				o.Points = append(o.Points, a.pts...)
			}
		}
		return o
	}
	quick := pick(func(int, []alt) int { return 0 })
	quick.Kind, quick.Label = "quickest", "Quickest"
	short := pick(func(_ int, alts []alt) int {
		best := 0
		for k, a := range alts {
			if a.km < alts[best].km {
				best = k
			}
		}
		return best
	})
	short.Kind, short.Label = "shortest", "Shortest"
	var out []Option
	if targetKm > 0 {
		out = append(out, c.closest(segs, pick, targetKm))
	}
	for _, o := range []Option{quick, short} {
		dup := false
		for _, e := range out {
			dup = dup || math.Abs(e.Km-o.Km) < 0.05
		}
		if !dup {
			out = append(out, o)
		}
	}
	return out, nil
}

// closest picks the alternative on each stretch so that the total is nearest the target length.
func (c *Client) closest(segs [][]alt, pick func(func(int, []alt) int) Option, target float64) Option {
	n := len(segs)
	choice := make([]int, n)
	best := math.Inf(1)
	bestChoice := make([]int, n)
	var walk func(i int, sum float64)
	walk = func(i int, sum float64) {
		if i == n {
			if d := math.Abs(sum - target); d < best-1e-9 {
				best = d
				copy(bestChoice, choice)
			}
			return
		}
		for k, a := range segs[i] {
			choice[i] = k
			walk(i+1, sum+a.km)
		}
	}
	walk(0, 0) // at most three roads on each of at most seven stretches: a couple of thousand totals
	o := pick(func(i int, _ []alt) int { return bestChoice[i] })
	o.Kind, o.Label = "closest", "Closest to the odometer"
	return o
}
