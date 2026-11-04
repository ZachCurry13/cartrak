// Package superchargers reads the public Supercharger list from supercharge.info and keeps the sites a non-Tesla car can use.
package superchargers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// DefaultURL is the community-kept list of every Tesla site. Only the list is downloaded; nothing about the car or the trip is sent.
const DefaultURL = "https://supercharge.info/service/supercharge/allSites"

// Site is one open Supercharger that other makes may charge at, trimmed to what trip planning needs.
type Site struct {
	Name   string  `json:"n"`
	Lat    float64 `json:"la"`
	Lon    float64 `json:"lo"`
	Kw     int     `json:"kw"`
	Stalls int     `json:"st"`
	NACS   bool    `json:"nacs"` // has NACS plugs: a Mach-E needs the NACS adapter
	Dock   bool    `json:"dock"` // has stalls with a built-in CCS connector (a "Magic Dock")
}

type raw struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Address struct {
		City, State string
	} `json:"address"`
	GPS struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	} `json:"gps"`
	StallCount int            `json:"stallCount"`
	Kw         int            `json:"powerKilowatt"`
	OtherEVs   bool           `json:"otherEVs"`
	Plugs      map[string]int `json:"plugs"`
}

// Parse keeps the open sites that let other cars charge and have NACS plugs or a CCS dock.
func Parse(r io.Reader) ([]Site, error) {
	var all []raw
	if err := json.NewDecoder(r).Decode(&all); err != nil {
		return nil, err
	}
	out := []Site{}
	for _, x := range all {
		if x.Status != "OPEN" || !x.OtherEVs || (x.GPS.Latitude == 0 && x.GPS.Longitude == 0) {
			continue
		}
		s := Site{Name: x.Name, Lat: x.GPS.Latitude, Lon: x.GPS.Longitude, Kw: x.Kw, Stalls: x.StallCount, NACS: x.Plugs["nacs"] > 0, Dock: x.Plugs["multi"] > 0 || x.Plugs["ccs1"] > 0}
		if s.NACS || s.Dock {
			out = append(out, s)
		}
	}
	return out, nil
}

// Fetch downloads and parses the list.
func Fetch(ctx context.Context, c *http.Client, url, agent string) ([]Site, error) {
	if c == nil {
		c = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", agent)
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("the Supercharger list answered %d", resp.StatusCode)
	}
	return Parse(io.LimitReader(resp.Body, 64<<20))
}

// Usable keeps the sites that fit what the car can plug into: "nacs" (NACS plugs, with the adapter), "dock" (CCS built in) or
// anything else for either.
func Usable(sites []Site, plugs string) []Site {
	out := make([]Site, 0, len(sites))
	for _, s := range sites {
		switch strings.ToLower(plugs) {
		case "nacs":
			if !s.NACS {
				continue
			}
		case "dock":
			if !s.Dock {
				continue
			}
		}
		out = append(out, s)
	}
	return out
}
