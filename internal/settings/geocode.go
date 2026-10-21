package settings

import (
	"fmt"
	"net/url"
	"strings"
)

// City lookup for the mileage report. On by default; only runs when an admin builds a report.
const (
	GeocodeEnabledKey = "CARTRAK_GEOCODE_ENABLED"
	GeocodeURLKey     = "CARTRAK_GEOCODE_URL"

	DefaultGeocodeURL = "https://nominatim.openstreetmap.org"
)

func init() {
	Defs = append(Defs,
		Def{Key: GeocodeEnabledKey, Label: "City names for trips and reports", Options: []Option{{"", "On (OpenStreetMap)"}, {"off", "Off: never send locations anywhere"}},
			Help: "Trips, the Day view, History and the mileage report show the town where each trip started and ended. CarTrak asks OpenStreetMap's Nominatim service from your server, a few places at a time in the background, with locations rounded to about 100 m, and remembers every answer so a place is only asked about once. Turn off to make no such requests; places you named still show."},
		Def{Key: GeocodeURLKey, Label: "City lookup service (advanced)", Placeholder: DefaultGeocodeURL,
			Help: "Only to use your own Nominatim server or another compatible service. Must be an https address (or http on localhost)."})
	validators[GeocodeEnabledKey] = func(val string) error {
		if val != "" && val != "off" {
			return fmt.Errorf("choose On or Off")
		}
		return nil
	}
	validators[GeocodeURLKey] = func(val string) error {
		if val == "" {
			return nil
		}
		u, err := url.Parse(val)
		local := err == nil && (strings.HasPrefix(u.Host, "localhost") || strings.HasPrefix(u.Host, "127.0.0.1"))
		if err != nil || u.Host == "" || (u.Scheme != "https" && !(u.Scheme == "http" && local)) {
			return fmt.Errorf("must be a full https:// address")
		}
		return nil
	}
}

// GeocodeEnabled is false only when an admin switched city lookup off.
func (s *Settings) GeocodeEnabled() bool { return s.Value(GeocodeEnabledKey) != "off" }

// GeocodeURL is the lookup service in use.
func (s *Settings) GeocodeURL() string {
	if v := s.Value(GeocodeURLKey); v != "" {
		return v
	}
	return DefaultGeocodeURL
}
