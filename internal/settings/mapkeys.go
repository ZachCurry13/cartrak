package settings

import (
	"fmt"
	"net/url"
	"strings"
)

// Map settings. Maps are on by default: CarTrak fetches OpenStreetMap tiles itself and passes them
// on, so a phone or computer never contacts the tile server.
const (
	MapEnabledKey     = "CARTRAK_MAP_ENABLED"
	MapTileURLKey     = "CARTRAK_MAP_TILE_URL"
	MapAttributionKey = "CARTRAK_MAP_ATTRIBUTION"

	DefaultTileURL     = "https://tile.openstreetmap.org/{z}/{x}/{y}.png"
	DefaultAttribution = "© OpenStreetMap contributors"
)

func init() {
	Defs = append(Defs,
		Def{Key: MapEnabledKey, Label: "Maps", Options: []Option{{"", "On (OpenStreetMap)"}, {"off", "Off: draw plain route lines"}},
			Help: "Trip maps use map tiles. CarTrak downloads the tiles on the server and keeps a copy on disk, so your phone never contacts the map provider. Turn off to draw plain route lines and make no map requests at all."},
		Def{Key: MapTileURLKey, Label: "Map tile server (advanced)", Placeholder: DefaultTileURL,
			Help: "Only to use a different tile provider. Must contain {z}, {x} and {y}. Follow that provider's usage rules; the default is OpenStreetMap's, which is meant for light use."},
		Def{Key: MapAttributionKey, Label: "Map credit (advanced)", Placeholder: DefaultAttribution,
			Help: "The credit line shown on maps. Change it together with the tile server, as the provider requires."})
	validators[MapEnabledKey] = func(val string) error {
		if val != "" && val != "off" {
			return fmt.Errorf("choose On or Off")
		}
		return nil
	}
	validators[MapTileURLKey] = func(val string) error {
		if val == "" {
			return nil
		}
		u, err := url.Parse(val)
		local := err == nil && (strings.HasPrefix(u.Host, "localhost") || strings.HasPrefix(u.Host, "127.0.0.1"))
		if err != nil || u.Host == "" || (u.Scheme != "https" && !(u.Scheme == "http" && local)) {
			return fmt.Errorf("must be a full https:// address")
		}
		for _, p := range []string{"{z}", "{x}", "{y}"} {
			if !strings.Contains(val, p) {
				return fmt.Errorf("must contain {z}, {x} and {y}")
			}
		}
		return nil
	}
}

// MapTileURL is the tile address in use.
func (s *Settings) MapTileURL() string {
	if v := s.Value(MapTileURLKey); v != "" {
		return v
	}
	return DefaultTileURL
}

// MapAttribution is the credit line in use.
func (s *Settings) MapAttribution() string {
	if v := s.Value(MapAttributionKey); v != "" {
		return v
	}
	return DefaultAttribution
}

// MapEnabled is false only when an admin switched maps off.
func (s *Settings) MapEnabled() bool { return s.Value(MapEnabledKey) != "off" }
