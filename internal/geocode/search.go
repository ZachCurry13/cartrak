package geocode

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Match is one place a typed name or address could mean.
type Match struct {
	Label   string   `json:"label"`   // the whole address, as the service writes it
	Name    string   `json:"name"`    // a business's name ("Flexible Circuits"), or the street for a plain address
	Address string   `json:"address"` // the short address: "1200 Main St, Maple Grove, MN"
	Kind    string   `json:"kind"`    // what it is, for a business or a landmark: "office", "bank", "restaurant"
	Lat     float64  `json:"lat"`
	Lon     float64  `json:"lon"`
	DistKm  *float64 `json:"distKm"` // how far from the place asked near, when there was one
}

// Search turns a typed address into up to five places. What was typed goes to the service and nowhere else: unlike the city
// lookups it is not remembered, because an address is something a person typed, not a rounded position. With a place to look near
// (nil for none), matches around it come first, so "Main St" finds the one in your own region, not the other side of the world;
// only a box about 300 miles wide is sent, rounded to a whole degree, never the exact place.
func (c *Client) Search(ctx context.Context, query string, near *[2]float64) ([]Match, error) {
	query = strings.TrimSpace(query)
	if len(query) < 3 || len(query) > 200 {
		return nil, fmt.Errorf("type a business name, or a street and a town")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.pace(ctx); err != nil {
		return nil, err
	}
	q := url.Values{"format": {"jsonv2"}, "limit": {"8"}, "addressdetails": {"1"}, "accept-language": {"en"}, "q": {query}}
	if near != nil {
		lat, lon := math.Round(near[0]), math.Round(near[1])
		q.Set("viewbox", fmt.Sprintf("%.0f,%.0f,%.0f,%.0f", lon-3, lat+3, lon+3, lat-3)) // left, top, right, bottom
		q.Set("bounded", "0")                                                            // a preference, not a limit: far-away places still turn up when nothing is near
	}
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
		DisplayName string            `json:"display_name"`
		Name        string            `json:"name"`
		Category    string            `json:"category"`
		Type        string            `json:"type"`
		Address     map[string]string `json:"address"`
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
		m := Match{Label: r.DisplayName, Name: r.Name, Address: shortAddress(r.Address), Kind: kindOf(r.Category, r.Type), Lat: lat, Lon: lon}
		if m.Name == "" {
			m.Name = strings.TrimSpace(strings.SplitN(r.DisplayName, ",", 2)[0])
		}
		if m.Address == "" {
			m.Address = r.DisplayName
		}
		if near != nil {
			d := haversineKm(near[0], near[1], lat, lon)
			m.DistKm = &d
		}
		out = append(out, m)
	}
	if near != nil { // several places with the same name: the closest first
		sort.SliceStable(out, func(i, j int) bool { return *out[i].DistKm < *out[j].DistKm })
	}
	return out, nil
}

// shortAddress is "1200 Main St, Maple Grove, MN": the street, the town and the state, without the county and the country.
func shortAddress(a map[string]string) string {
	street := strings.TrimSpace(strings.TrimSpace(a["house_number"]) + " " + strings.TrimSpace(a["road"]))
	state := a["state"]
	if st := pickState(a); st != "" {
		state = st
	}
	var parts []string
	for _, p := range []string{street, pickCity(a), state} {
		if p = strings.TrimSpace(p); p != "" {
			parts = append(parts, p)
		}
	}
	return strings.Join(parts, ", ")
}

// kindOf names what a business or landmark is. Streets, towns and the like need no label.
func kindOf(category, typ string) string {
	switch category {
	case "", "place", "boundary", "highway", "building", "landuse":
		return ""
	}
	return strings.ReplaceAll(typ, "_", " ")
}

func haversineKm(lat1, lon1, lat2, lon2 float64) float64 {
	const rad = math.Pi / 180
	a := math.Sin((lat2-lat1)*rad/2)*math.Sin((lat2-lat1)*rad/2) + math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin((lon2-lon1)*rad/2)*math.Sin((lon2-lon1)*rad/2)
	return 6371 * 2 * math.Asin(math.Sqrt(a))
}
