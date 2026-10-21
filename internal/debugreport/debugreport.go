// Package debugreport writes the plain-text report an owner can copy and send when something looks wrong. It lists
// what the car reported, the trips and charging sessions CarTrak made from it, and the alerts it sent. It never
// includes positions, sign-ins, keys or the VIN.
package debugreport

import (
	"fmt"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

// Input is everything the report is built from.
type Input struct {
	Version   string
	Now       time.Time
	Loc       *time.Location
	From      time.Time
	Vehicle   string
	Linked    bool
	LastOK    time.Time
	LastErr   string
	Metric    bool // distances and speeds in kilometres; otherwise miles
	Rows      []store.DebugRow
	Locks     map[int64][]string // raw lock and door fields, by reading time
	Fields    []string           // every field of the latest reading as "name = value", positions and identifiers already removed
	Trips     []*store.Trip
	Polls     []PollInfo       // the latest polls CarTrak made (since it last started)
	Decisions []store.LogEntry // blips thrown away, charges picked up again, alerts held back, and why
	Settings  []string         // the settings that shape behaviour, without secrets or positions
	Curves    map[int64]string // by charging session id: the power at 20, 40, 60 and 80 percent
	CostFrom  map[int64]string // by charging session id: where its cost came from and the price per kWh
	Days      []string         // trip-away stretches and how each day's base was settled, no positions
	ClientApp []string         // the phone's loaded app version and its recent errors, added by the page
	Points    map[int64][3]int // by trip id: readings the car sent, how many had a position, and phone GPS points
	Charges   []*store.Charge
	Alerts    []store.AlertLogEntry
	MaxLines  int
}

func fp(p *float64, f string) string {
	if p == nil {
		return "-"
	}
	return fmt.Sprintf(f, *p)
}

// PollInfo is one poll of Ford.
type PollInfo struct {
	TS      int64
	Result  string
	NextSec int
	Detail  string
}

// kwText is the power of a reading: the car's own figure, or one worked out from the pack (marked ~).
func kwText(r store.DebugRow) string {
	if r.ChargeKw != nil && *r.ChargeKw > 0.2 {
		return fmt.Sprintf("%.1f", *r.ChargeKw)
	}
	if r.PackKw != nil && *r.PackKw > 0.2 {
		return fmt.Sprintf("~%.1f", *r.PackKw)
	}
	return fp(r.ChargeKw, "%.1f")
}

func lock(b *bool) string {
	switch {
	case b == nil:
		return "?"
	case *b:
		return "locked"
	default:
		return "UNLOCKED"
	}
}

func dash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
