// Package fleet runs every car CarTrak follows: one database, poller, trip detector, alert engine and server per car,
// all sharing one core database of people, sign-ins, notification devices and installation-wide settings.
package fleet

import (
	"strings"

	"github.com/ZachCurry13/cartrak/internal/store"
)

// globalKeys belong to the whole installation: the notification identity, the update check, the connector, the
// remote-access settings. They always live in the core database.
var globalKeys = []string{"vapid_private", "vapid_public", "update_check", "onboarding_dismissed",
	"connector_bin", "connector_prev", "connector_failed",
	"setting:CARTRAK_TUNNEL_TOKEN", "setting:CARTRAK_PUBLIC_ADDRESS", "setting:CARTRAK_REMOTE_ENABLED"}

// inheritedKeys are settings every car starts with the same value for (Ford's app, the map and lookup services, the time
// zone, the currency). A car can set its own; until it does it uses the installation's.
var inheritedKeys = []string{"setting:FORD_CLIENT_ID", "setting:FORD_CLIENT_SECRET", "setting:FORD_REDIRECT_URL", "setting:FORD_AUTH_URL",
	"setting:CARTRAK_TZ", "setting:CARTRAK_GEOCODE_ENABLED", "setting:CARTRAK_GEOCODE_URL", "setting:CARTRAK_MAP_ENABLED",
	"setting:CARTRAK_MAP_TILE_URL", "setting:CARTRAK_MAP_ATTRIBUTION", "setting:CARTRAK_ROUTE_LOOKUP", "setting:CARTRAK_ROUTE_URL",
	"setting:CARTRAK_WEATHER_LOOKUP", "setting:CARTRAK_CURRENCY", "setting:CARTRAK_EIA_KEY", "setting:CARTRAK_GAS_ADJUST"}

// GlobalKeys is what ImportIdentity copies from the first car's database: the global keys and the inherited defaults.
func GlobalKeys() []string { return append(append([]string{}, globalKeys...), inheritedKeys...) }

func in(list []string, k string) bool {
	for _, x := range list {
		if x == k {
			return true
		}
	}
	return false
}

// CarKV is the key-value store one car's settings and state use. Most keys are the car's own. Global keys are the
// installation's, in the core database. Inherited keys read the car's own value first and the installation's after
// it; the first car also writes them to the installation, so they are the starting point for the cars added later.
type CarKV struct {
	Car     *store.Store
	Core    *store.Store
	Primary bool
}

func (k *CarKV) GetKV(key string) string {
	switch {
	case in(globalKeys, key) || strings.HasPrefix(key, "global:"):
		return k.Core.GetKV(key)
	case in(inheritedKeys, key):
		if v := k.Car.GetKV(key); v != "" {
			return v
		}
		return k.Core.GetKV(key)
	}
	return k.Car.GetKV(key)
}

func (k *CarKV) SetKV(key, val string) error {
	switch {
	case in(globalKeys, key) || strings.HasPrefix(key, "global:"):
		return k.Core.SetKV(key, val)
	case in(inheritedKeys, key):
		if k.Primary {
			if err := k.Core.SetKV(key, val); err != nil {
				return err
			}
		}
		return k.Car.SetKV(key, val)
	}
	return k.Car.SetKV(key, val)
}
