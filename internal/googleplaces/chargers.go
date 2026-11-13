package googleplaces

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/ZachCurry13/cartrak/internal/geocode"
)

// NearbyURL is Google's Places API (New) nearby search, used for chargers and how many of their plugs are free.
const NearbyURL = "https://places.googleapis.com/v1/places:searchNearby"

// Charger is a charging place with its plugs. Available is how many plugs the network reports free, when it reports that.
type Charger struct {
	Name       string      `json:"name"`
	Address    string      `json:"address"`
	Lat        float64     `json:"lat"`
	Lon        float64     `json:"lon"`
	DistKm     float64     `json:"distKm"`
	Connectors []Connector `json:"connectors"`
}

// Connector is one kind of plug at a charger.
type Connector struct {
	Type         string  `json:"type"` // "CCS", "Tesla (NACS)", "J1772", ...
	Kw           float64 `json:"kw"`
	Count        int     `json:"count"`
	Available    *int    `json:"available"`    // nil when the network does not report it
	OutOfService *int    `json:"outOfService"` // nil when the network does not report it
	Updated      int64   `json:"updated,omitempty"`
}

var plugNames = map[string]string{"EV_CONNECTOR_TYPE_CCS_COMBO_1": "CCS", "EV_CONNECTOR_TYPE_CCS_COMBO_2": "CCS2", "EV_CONNECTOR_TYPE_TESLA": "Tesla (NACS)",
	"EV_CONNECTOR_TYPE_J1772": "J1772", "EV_CONNECTOR_TYPE_CHADEMO": "CHAdeMO", "EV_CONNECTOR_TYPE_TYPE_2": "Type 2", "EV_CONNECTOR_TYPE_UNSPECIFIED_WALL_OUTLET": "Wall outlet"}

// Chargers lists the chargers within radiusM of a place, nearest first (at most ten). Only that position and the key are sent.
func Chargers(ctx context.Context, hc *http.Client, base, key string, lat, lon, radiusM float64) ([]Charger, error) {
	if key == "" {
		return nil, fmt.Errorf("the Google Places key is needed")
	}
	if base == "" {
		base = NearbyURL
	}
	if hc == nil {
		hc = &http.Client{Timeout: 15 * time.Second}
	}
	var in struct {
		IncludedTypes  []string `json:"includedTypes"`
		MaxResultCount int      `json:"maxResultCount"`
		RankPreference string   `json:"rankPreference"`
		Location       struct {
			Circle struct {
				Center struct {
					Latitude  float64 `json:"latitude"`
					Longitude float64 `json:"longitude"`
				} `json:"center"`
				Radius float64 `json:"radius"`
			} `json:"circle"`
		} `json:"locationRestriction"`
	}
	in.IncludedTypes, in.MaxResultCount, in.RankPreference = []string{"electric_vehicle_charging_station"}, 10, "DISTANCE"
	in.Location.Circle.Center.Latitude, in.Location.Circle.Center.Longitude, in.Location.Circle.Radius = lat, lon, radiusM
	body, _ := json.Marshal(in)
	req, err := http.NewRequestWithContext(ctx, "POST", base, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Goog-Api-Key", key)
	req.Header.Set("X-Goog-FieldMask", "places.displayName,places.formattedAddress,places.location,places.evChargeOptions")
	resp, err := hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Google could not be reached")
	}
	defer resp.Body.Close()
	var out struct {
		Places []struct {
			DisplayName struct {
				Text string `json:"text"`
			} `json:"displayName"`
			FormattedAddress string `json:"formattedAddress"`
			Location         struct {
				Latitude  float64 `json:"latitude"`
				Longitude float64 `json:"longitude"`
			} `json:"location"`
			EV struct {
				Aggregation []struct {
					Type         string  `json:"type"`
					MaxKw        float64 `json:"maxChargeRateKw"`
					Count        int     `json:"count"`
					Updated      string  `json:"availabilityLastUpdateTime"`
					Available    *int    `json:"availableCount"`
					OutOfService *int    `json:"outOfServiceCount"`
				} `json:"connectorAggregation"`
			} `json:"evChargeOptions"`
		} `json:"places"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	_ = json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&out)
	if resp.StatusCode != 200 {
		msg := out.Error.Message
		if msg == "" {
			msg = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("Google answered: %s", msg)
	}
	list := []Charger{}
	for _, p := range out.Places {
		c := Charger{Name: p.DisplayName.Text, Address: p.FormattedAddress, Lat: p.Location.Latitude, Lon: p.Location.Longitude,
			DistKm: geocode.DistanceKm(lat, lon, p.Location.Latitude, p.Location.Longitude), Connectors: []Connector{}}
		for _, a := range p.EV.Aggregation {
			name := plugNames[a.Type]
			if name == "" {
				name = strings.ToLower(strings.ReplaceAll(strings.TrimPrefix(a.Type, "EV_CONNECTOR_TYPE_"), "_", " "))
				if name != "" {
					name = strings.ToUpper(name[:1]) + name[1:]
				}
			}
			k := Connector{Type: name, Kw: a.MaxKw, Count: a.Count, Available: a.Available, OutOfService: a.OutOfService}
			if t, err := time.Parse(time.RFC3339, a.Updated); err == nil {
				k.Updated = t.Unix()
			}
			c.Connectors = append(c.Connectors, k)
		}
		sort.SliceStable(c.Connectors, func(i, j int) bool { return c.Connectors[i].Kw > c.Connectors[j].Kw })
		list = append(list, c)
	}
	sort.SliceStable(list, func(i, j int) bool { return list[i].DistKm < list[j].DistKm })
	return list, nil
}
