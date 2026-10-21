// Package share prepares a route for a public link: trimming the ends so the start and finish (often home) are
// not shown, thinning it, and working out which map tiles the link may fetch.
package share

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// TrimMeters is how much of each end of a route is left out of a trimmed link (about a quarter of a mile).
const TrimMeters = 400.0

func meters(a, b [2]float64) float64 {
	const r = 6371000.0
	la1, la2 := a[0]*math.Pi/180, b[0]*math.Pi/180
	dLat, dLon := (b[0]-a[0])*math.Pi/180, (b[1]-a[1])*math.Pi/180
	h := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(la1)*math.Cos(la2)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 2 * r * math.Asin(math.Sqrt(h))
}

// Trim removes the first and last `m` meters of the path. A path too short to keep anything comes back empty.
func Trim(pts [][2]float64, m float64) [][2]float64 {
	if len(pts) < 2 {
		return nil
	}
	start := 0
	for acc := 0.0; start < len(pts)-1 && acc < m; start++ {
		acc += meters(pts[start], pts[start+1])
	}
	end := len(pts) - 1
	for acc := 0.0; end > start && acc < m; end-- {
		acc += meters(pts[end], pts[end-1])
	}
	if end-start < 1 {
		return nil
	}
	return append([][2]float64(nil), pts[start:end+1]...)
}

// Thin keeps at most max points, always the first and the last, and rounds to about a meter.
func Thin(pts [][2]float64, max int) [][2]float64 {
	if max < 2 {
		max = 2
	}
	keep := pts
	if len(pts) > max {
		keep = make([][2]float64, 0, max)
		step := float64(len(pts)-1) / float64(max-1)
		for i := 0; i < max; i++ {
			keep = append(keep, pts[int(float64(i)*step+0.5)])
		}
	}
	out := make([][2]float64, len(keep))
	for i, p := range keep {
		out[i] = [2]float64{math.Round(p[0]*1e5) / 1e5, math.Round(p[1]*1e5) / 1e5}
	}
	return out
}

// BBox is "minLat,minLon,maxLat,maxLon" of all the routes, or "" when there are no points.
func BBox(routes ...[][2]float64) string {
	first := true
	var a, b, c, d float64
	for _, r := range routes {
		for _, p := range r {
			if first {
				a, b, c, d, first = p[0], p[1], p[0], p[1], false
				continue
			}
			a, b, c, d = math.Min(a, p[0]), math.Min(b, p[1]), math.Max(c, p[0]), math.Max(d, p[1])
		}
	}
	if first {
		return ""
	}
	return fmt.Sprintf("%.5f,%.5f,%.5f,%.5f", a, b, c, d)
}

func parseBBox(s string) (v [4]float64, ok bool) {
	parts := strings.Split(s, ",")
	if len(parts) != 4 {
		return v, false
	}
	for i, p := range parts {
		f, err := strconv.ParseFloat(p, 64)
		if err != nil {
			return v, false
		}
		v[i] = f
	}
	return v, true
}

func tileBounds(z, x, y int) (minLat, minLon, maxLat, maxLon float64) {
	n := math.Pow(2, float64(z))
	lon := func(x float64) float64 { return x/n*360 - 180 }
	lat := func(y float64) float64 { return math.Atan(math.Sinh(math.Pi*(1-2*y/n))) * 180 / math.Pi }
	return lat(float64(y + 1)), lon(float64(x)), lat(float64(y)), lon(float64(x + 1))
}

// TileAllowed says whether a link may fetch this map tile: it must be near the shared route (inside its
// bounding box with a margin of a few tiles) and not zoomed in closer than street level. A link cannot be
// used as a general map proxy.
func TileAllowed(bbox string, z, x, y int) bool {
	if z < 0 || z > 16 {
		return false
	}
	b, ok := parseBBox(bbox)
	if !ok {
		return false
	}
	if z <= 6 {
		return true // the world and country views reveal nothing
	}
	minLat, minLon, maxLat, maxLon := tileBounds(z, x, y)
	pad := 3 * (maxLon - minLon) // a few tiles of margin, so panning a little still works
	return maxLat >= b[0]-pad && minLat <= b[2]+pad && maxLon >= b[1]-pad && minLon <= b[3]+pad
}
