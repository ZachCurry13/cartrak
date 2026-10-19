// Package debugreport writes the plain-text report an owner can copy and send when something looks wrong. It lists
// what the car reported, the trips and charging sessions CarTrak made from it, and the alerts it sent. It never
// includes positions, sign-ins, keys or the VIN.
package debugreport

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

// Input is everything the report is built from.
type Input struct {
	Version  string
	Now      time.Time
	Loc      *time.Location
	From     time.Time
	Vehicle  string
	Linked   bool
	LastOK   time.Time
	LastErr  string
	Rows     []store.DebugRow
	Locks    map[int64][]string // raw lock and door fields, by reading time
	Fields   []string           // every field of the latest reading as "name = value", positions and identifiers already removed
	Trips    []*store.Trip
	Charges  []*store.Charge
	Alerts   []store.AlertLogEntry
	MaxLines int
}

func fp(p *float64, f string) string {
	if p == nil {
		return "-"
	}
	return fmt.Sprintf(f, *p)
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

// Build writes the report.
func Build(in Input) string {
	var b strings.Builder
	at := func(ts int64) string { return time.Unix(ts, 0).In(in.Loc).Format("01-02 15:04:05") }
	fmt.Fprintf(&b, "CarTrak debug report\nversion %s, made %s (%s)\nvehicle: %s\n", in.Version, in.Now.In(in.Loc).Format("2006-01-02 15:04:05"), in.Loc, dash(in.Vehicle))
	last := "never"
	if !in.LastOK.IsZero() {
		last = in.LastOK.In(in.Loc).Format("01-02 15:04:05")
	}
	fmt.Fprintf(&b, "Ford link: linked=%v, last answer %s, last error: %s\n", in.Linked, last, dash(in.LastErr))
	fmt.Fprintf(&b, "window: from %s, %d readings\n\n", in.From.In(in.Loc).Format("01-02 15:04"), len(in.Rows))

	b.WriteString("READINGS (time | ignition | speed km/h | plug | charge state | kW | battery % | odometer km | lock)\n")
	max := in.MaxLines
	if max <= 0 {
		max = 700
	}
	type line struct {
		ts   int64
		text string
	}
	var lines []line
	var prevKey string
	repeat, repeatFrom := 0, int64(0)
	flush := func(end int64) {
		if repeat > 0 {
			lines = append(lines, line{end, fmt.Sprintf("  ... the same %d more time(s) until %s", repeat, at(end))})
		}
		repeat = 0
	}
	for _, r := range in.Rows {
		key := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s", dash(r.Ignition), fp(r.SpeedKph, "%.0f"), dash(r.Plug), dash(r.ChargeState), fp(r.ChargeKw, "%.1f"), fp(r.SoC, "%.0f"), fp(r.OdometerKm, "%.0f"), lock(r.Locked))
		if key == prevKey {
			repeat++
			repeatFrom = r.TS
			continue
		}
		flush(repeatFrom)
		prevKey = key
		lines = append(lines, line{r.TS, at(r.TS) + " | " + strings.ReplaceAll(key, "|", " | ")})
	}
	flush(repeatFrom)
	if len(lines) > max { // keep the most recent
		lines = append([]line{{0, fmt.Sprintf("(%d older lines left out)", len(lines)-max)}}, lines[len(lines)-max:]...)
	}
	for _, l := range lines {
		b.WriteString(l.text + "\n")
	}

	b.WriteString("\nLOCK AND DOOR FIELDS as the car sent them, each time the lock state changed and for the latest reading\n")
	var tss []int64
	for ts := range in.Locks {
		tss = append(tss, ts)
	}
	sort.Slice(tss, func(i, j int) bool { return tss[i] < tss[j] })
	if len(tss) > 40 {
		tss = tss[len(tss)-40:]
	}
	for _, ts := range tss {
		fmt.Fprintf(&b, "%s\n", at(ts))
		for _, kv := range in.Locks[ts] {
			fmt.Fprintf(&b, "    %s\n", kv)
		}
	}
	if len(tss) == 0 {
		b.WriteString("(none in this window)\n")
	}

	fmt.Fprintf(&b, "\nEVERY FIELD IN THE LATEST READING (%d)\n", len(in.Fields))
	for _, f := range in.Fields {
		b.WriteString("    " + f + "\n")
	}

	b.WriteString("\nTRIPS (id | start | end | km | battery start>end | energy kWh start>end | open | category)\n")
	for _, t := range in.Trips {
		end := "open"
		if t.EndTS != nil {
			end = at(*t.EndTS)
		}
		fmt.Fprintf(&b, "%d | %s | %s | %.1f | %s>%s | %s>%s | %v | %s\n", t.ID, at(t.StartTS), end, t.DistanceKm, fp(t.StartSoC, "%.0f"), fp(t.EndSoC, "%.0f"), fp(t.StartKwh, "%.1f"), fp(t.EndKwh, "%.1f"), t.Open, t.Category)
	}
	if len(in.Trips) == 0 {
		b.WriteString("(none)\n")
	}

	b.WriteString("\nCHARGING (id | start | end | battery start>end | energy kWh start>end | peak kW | charger kWh | open)\n")
	for _, c := range in.Charges {
		end := "open"
		if c.EndTS != nil {
			end = at(*c.EndTS)
		}
		fmt.Fprintf(&b, "%d | %s | %s | %s>%s | %s>%s | %.1f | %s | %v\n", c.ID, at(c.StartTS), end, fp(c.StartSoC, "%.0f"), fp(c.EndSoC, "%.0f"), fp(c.StartKwh, "%.1f"), fp(c.EndKwh, "%.1f"), c.PeakKw, fp(c.ChargerKwh, "%.1f"), c.Open)
	}
	if len(in.Charges) == 0 {
		b.WriteString("(none)\n")
	}

	b.WriteString("\nALERTS SENT (time | kind | title)\n")
	for _, a := range in.Alerts {
		if time.Unix(a.TS, 0).Before(in.From) {
			continue
		}
		fmt.Fprintf(&b, "%s | %s | %s\n", at(a.TS), a.Kind, a.Title)
	}
	return b.String()
}
