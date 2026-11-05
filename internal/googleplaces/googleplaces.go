// Package googleplaces finds a business by name with Google's Places API (Text Search). It is optional: with no key CarTrak keeps
// using the map's own search, which does not list many small businesses. Only the searched name and Home's rough area leave the server.
package googleplaces

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ZachCurry13/cartrak/internal/geocode"
)

// DefaultURL is Google's Places API (New) text search.
const DefaultURL = "https://places.googleapis.com/v1/places:searchText"

type request struct {
	TextQuery    string `json:"textQuery"`
	RegionCode   string `json:"regionCode,omitempty"`
	PageSize     int    `json:"pageSize"`
	LocationBias *struct {
		Circle struct {
			Center struct {
				Latitude  float64 `json:"latitude"`
				Longitude float64 `json:"longitude"`
			} `json:"center"`
			Radius float64 `json:"radius"`
		} `json:"circle"`
	} `json:"locationBias,omitempty"`
}

type response struct {
	Places []struct {
		DisplayName struct {
			Text string `json:"text"`
		} `json:"displayName"`
		FormattedAddress string `json:"formattedAddress"`
		Location         struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		} `json:"location"`
		PrimaryTypeDisplayName struct {
			Text string `json:"text"`
		} `json:"primaryTypeDisplayName"`
	} `json:"places"`
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Search returns up to five places for the text, nearest Home first when Home is known. region is a country code such as "US".
func Search(ctx context.Context, hc *http.Client, base, key, query, region string, near *[2]float64) ([]geocode.Match, error) {
	query = strings.TrimSpace(query)
	if key == "" || len(query) < 2 || len(query) > 200 {
		return nil, fmt.Errorf("a key and a name are needed")
	}
	if base == "" {
		base = DefaultURL
	}
	if hc == nil {
		hc = &http.Client{Timeout: 15 * time.Second}
	}
	in := request{TextQuery: query, RegionCode: region, PageSize: 5}
	if near != nil {
		in.LocationBias = &struct {
			Circle struct {
				Center struct {
					Latitude  float64 `json:"latitude"`
					Longitude float64 `json:"longitude"`
				} `json:"center"`
				Radius float64 `json:"radius"`
			} `json:"circle"`
		}{}
		in.LocationBias.Circle.Center.Latitude, in.LocationBias.Circle.Center.Longitude, in.LocationBias.Circle.Radius = near[0], near[1], 50000
	}
	body, _ := json.Marshal(in)
	req, err := http.NewRequestWithContext(ctx, "POST", base, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Goog-Api-Key", key)
	req.Header.Set("X-Goog-FieldMask", "places.displayName,places.formattedAddress,places.location,places.primaryTypeDisplayName")
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out response
	_ = json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&out)
	if resp.StatusCode != 200 {
		msg := out.Error.Message
		if msg == "" {
			msg = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("Google answered: %s", msg)
	}
	var found []geocode.Match
	for _, p := range out.Places {
		m := geocode.Match{Label: p.DisplayName.Text + ", " + p.FormattedAddress, Name: p.DisplayName.Text, Address: p.FormattedAddress,
			Kind: strings.ToLower(p.PrimaryTypeDisplayName.Text), Lat: p.Location.Latitude, Lon: p.Location.Longitude}
		if near != nil {
			d := geocode.DistanceKm(near[0], near[1], m.Lat, m.Lon)
			m.DistKm = &d
		}
		found = append(found, m)
	}
	return found, nil
}

// Ping asks for one common place, to check the key works.
func Ping(ctx context.Context, hc *http.Client, base, key string) error {
	_, err := Search(ctx, hc, base, key, "coffee", "US", nil)
	return err
}
