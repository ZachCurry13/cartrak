// Package afdc reads Electrify America's chargers from the US government's Alternative Fuels Data Center list.
package afdc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/ZachCurry13/cartrak/internal/superchargers"
)

// DefaultURL is the station list.
const DefaultURL = "https://developer.nlr.gov/api/alt-fuel-stations/v1.json"

// eaKw is what an Electrify America charger is assumed to give: the list has no power, and its chargers are 150 to 350 kW.
const eaKw = 150

type reply struct {
	Stations []struct {
		Name       string   `json:"station_name"`
		Lat        float64  `json:"latitude"`
		Lon        float64  `json:"longitude"`
		Fast       int      `json:"ev_dc_fast_num"`
		Connectors []string `json:"ev_connector_types"`
	} `json:"fuel_stations"`
}

// Parse keeps the chargers with a CCS connector and at least one fast port.
func Parse(r io.Reader) ([]superchargers.Site, error) {
	var x reply
	if err := json.NewDecoder(r).Decode(&x); err != nil {
		return nil, err
	}
	out := []superchargers.Site{}
	for _, s := range x.Stations {
		ccs := false
		for _, c := range s.Connectors {
			ccs = ccs || c == "J1772COMBO"
		}
		if ccs && s.Fast > 0 && !(s.Lat == 0 && s.Lon == 0) {
			out = append(out, superchargers.Site{Name: "Electrify America: " + s.Name, Lat: s.Lat, Lon: s.Lon, Kw: eaKw, Stalls: s.Fast, Dock: true, Net: "Electrify America"})
		}
	}
	return out, nil
}

// Fetch downloads the open, public Electrify America chargers in the US. The key goes to the service as the list requires.
func Fetch(ctx context.Context, c *http.Client, base, key string) ([]superchargers.Site, error) {
	if c == nil {
		c = http.DefaultClient
	}
	q := url.Values{"api_key": {key}, "fuel_type": {"ELEC"}, "ev_network": {"Electrify America"}, "status": {"E"}, "access": {"public"}, "country": {"US"}, "limit": {"all"}}
	req, err := http.NewRequestWithContext(ctx, "GET", base+"?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("the charger list answered %d (check the key)", resp.StatusCode)
	}
	return Parse(io.LimitReader(resp.Body, 64<<20))
}

// Ping asks the list for one station, to check the key works.
func Ping(ctx context.Context, c *http.Client, base, key string) error {
	if c == nil {
		c = http.DefaultClient
	}
	q := url.Values{"api_key": {key}, "fuel_type": {"ELEC"}, "ev_network": {"Electrify America"}, "limit": {"1"}}
	req, err := http.NewRequestWithContext(ctx, "GET", base+"?"+q.Encode(), nil)
	if err != nil {
		return err
	}
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case 200:
		return nil
	case 401, 403:
		return fmt.Errorf("the list did not accept the key (%d)", resp.StatusCode)
	}
	return fmt.Errorf("the list answered %d", resp.StatusCode)
}
