// Package ford talks to Ford's cloud. Two interchangeable providers exist:
// "fordconnect" (official partner API) and "fordpass" (the FordPass app's API).
package ford

import (
	"context"
	"encoding/json"
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
