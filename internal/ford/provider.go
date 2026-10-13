// Package ford talks to Ford's cloud. Two interchangeable providers exist:
// "fordconnect" (official partner API) and "fordpass" (the FordPass app's API).
package ford

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// State is a provider-neutral vehicle reading. Pointer fields are nil when
// the provider did not report them. Raw always holds the untouched response.
type State struct {
	TS          time.Time
	OdometerKm  *float64
	SoCPct      *float64
	RangeKm     *float64
	Lat, Lon    *float64
	SpeedKph    *float64
	Ignition    string // Off, Accessory, On, Run, Start...
	ChargeState string // e.g. NotReady, Charging, Complete
	Plug        string // Disconnected / Connected
	ChargeKw    *float64
	OutsideC    *float64
	Battery12V  *float64
	Tires       [4]*float64 // FL FR RL RR, kPa
	FuelPct     *float64    // fuel tank level, gas and hybrid cars
	OilLifePct  *float64    // remaining oil life, cars with an engine
	HVVolts     *float64    // traction battery voltage, electrified cars
	CapacityKwh *float64    // estimated traction battery capacity
	Locked      *bool
	Extra       map[string]any // every other field the provider returned
	Raw         json.RawMessage
}

type Provider interface {
	Name() string
	// Ready reports whether credentials/tokens are in place.
	Ready() bool
	// Fetch returns the current vehicle state.
	Fetch(ctx context.Context) (*State, error)
}

func ptr[T any](v T) *T { return &v }

// RateLimitError means Ford answered "too many requests" (HTTP 429).
// After is how long Ford asked us to wait (zero if it did not say).
type RateLimitError struct{ After time.Duration }

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("ford rate limit (retry after %s)", e.After)
}
