package debugreport

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// Build writes the report.
func Build(in Input) string {
	var b strings.Builder
	dunit, sunit, per := "mi", "mph", 1.0/1.609344 // distances, speeds, and what to multiply kilometers by
	if in.Metric {
		dunit, sunit, per = "km", "km/h", 1
	}
	conv := func(p *float64, f string) string {
		if p == nil {
			return "-"
		}
		return fmt.Sprintf(f, *p*per)
	}
	at := func(ts int64) string { return time.Unix(ts, 0).In(in.Loc).Format("01-02 15:04:05") }
	fmt.Fprintf(&b, "CarTrak debug report\nversion %s, made %s (%s)\nvehicle: %s\n", in.Version, in.Now.In(in.Loc).Format("2006-01-02 15:04:05"), in.Loc, dash(in.Vehicle))
	last := "never"
	if !in.LastOK.IsZero() {
		last = in.LastOK.In(in.Loc).Format("01-02 15:04:05")
	}
	fmt.Fprintf(&b, "Ford link: linked=%v, last answer %s, last error: %s\n", in.Linked, last, dash(in.LastErr))
	fmt.Fprintf(&b, "window: from %s, %d readings\n\n", in.From.In(in.Loc).Format("01-02 15:04"), len(in.Rows))

	b.WriteString("READINGS (time | ignition | speed " + sunit + " | plug | charge state | kW (~ worked out from the pack, which fast chargers need) | kWh left | battery % | odometer " + dunit + " | lock)\n")
	max := in.MaxLines
	if max <= 0 {
		max = 1100
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
		key := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s|%s", dash(r.Ignition), conv(r.SpeedKph, "%.0f"), dash(r.Plug), dash(r.ChargeState), kwText(r), fp(r.EnergyKwh, "%.1f"), fp(r.SoC, "%.0f"), conv(r.OdometerKm, "%.0f"), lock(r.Locked))
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

	b.WriteString("\nLOCKS AND DOORS each time the lock state changed and at the latest reading (the raw fields are in the full list below)\n")
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

	b.WriteString("\nTRIPS (id | start | end | " + dunit + " | battery start>end | energy kWh start>end | open | category; then a line of readings, different car positions, phone points)\n")
	for _, t := range in.Trips {
		end := "open"
		if t.EndTS != nil {
			end = at(*t.EndTS)
		}
		fmt.Fprintf(&b, "%d | %s | %s | %.1f | %s>%s | %s>%s | %v | %s%s\n", t.ID, at(t.StartTS), end, t.DistanceKm*per, fp(t.StartSoC, "%.0f"), fp(t.EndSoC, "%.0f"), fp(t.StartKwh, "%.1f"), fp(t.EndKwh, "%.1f"), t.Open, t.Category, distanceNote(t, per, dunit))
		if p, ok := in.Points[t.ID]; ok {
			fmt.Fprintf(&b, "    positions: %d readings, %d different car positions (the car repeats its last one), %d phone GPS points\n", p[0], p[1], p[2])
		}
	}
	if len(in.Trips) == 0 {
		b.WriteString("(none)\n")
	}

	b.WriteString("\nCHARGING (id | start | end | battery start>end | energy kWh start>end | peak kW | charger kWh | power kWh (the reported power added up) | open; then charger type, what you paid and the power at 20/40/60/80 percent)\n")
	for _, c := range in.Charges {
		end := "open"
		if c.EndTS != nil {
			end = at(*c.EndTS)
		}
		fmt.Fprintf(&b, "%d | %s | %s | %s>%s | %s>%s | %.1f | %s | %s | %v\n", c.ID, at(c.StartTS), end, fp(c.StartSoC, "%.0f"), fp(c.EndSoC, "%.0f"), fp(c.StartKwh, "%.1f"), fp(c.EndKwh, "%.1f"), c.PeakKw, fp(c.ChargerKwh, "%.1f"), fp(c.PowerKwh, "%.1f"), c.Open)
		if c.Station != "" || c.ActualCost != nil {
			fmt.Fprintf(&b, "    charger type %s, paid %s\n", dash(c.Station), fp(c.ActualCost, "%.2f"))
		}
		if cv, ok := in.Curves[c.ID]; ok {
			fmt.Fprintf(&b, "    power: %s\n", cv)
		}
		if cf, ok := in.CostFrom[c.ID]; ok {
			fmt.Fprintf(&b, "    cost: %s\n", cf)
		}
	}
	if len(in.Charges) == 0 {
		b.WriteString("(none)\n")
	}

	if len(in.Fuel) > 0 {
		b.WriteString("\nFUEL (tank size learned from fill-ups; fill-ups with the gauge before and after; the tank gauge over time)\n")
		for _, l := range in.Fuel {
			b.WriteString("    " + l + "\n")
		}
	}

	if len(in.Slowest) > 0 {
		b.WriteString("\nSLOWEST REQUESTS (average and slowest answer since CarTrak last started; a slow page is slow because of one of these)\n")
		for _, l := range in.Slowest {
			b.WriteString("    " + l + "\n")
		}
	}

	b.WriteString("\nSERVER ERRORS (requests that hit a bug since CarTrak last started)\n")
	if len(in.ServerErrors) == 0 {
		b.WriteString("(none)\n")
	}
	for _, l := range in.ServerErrors {
		b.WriteString("    " + l + "\n")
	}

	if len(in.Days) > 0 {
		b.WriteString("\nTIME AWAY (stretches and each day's base; no positions)\n")
		for _, l := range in.Days {
			b.WriteString("    " + l + "\n")
		}
	}

	b.WriteString("\nSETTINGS THAT SHAPE BEHAVIOR (no secrets, no positions)\n")
	for _, l := range in.Settings {
		b.WriteString("    " + l + "\n")
	}

	b.WriteString("\nPOLLING (how often CarTrak asked Ford, since it last started)\n")
	b.WriteString(pollSummary(in.Polls, at))

	b.WriteString("\nDECISIONS CARTRAK MADE, AND CHANGES MADE BY HAND, THAT LEAVE NO OTHER TRACE (time | kind | what)\n")
	anyDecision := false
	for _, d := range in.Decisions {
		if time.Unix(d.TS, 0).Before(in.From) {
			continue
		}
		anyDecision = true
		fmt.Fprintf(&b, "%s | %s | %s\n", at(d.TS), d.Kind, d.Note)
	}
	if !anyDecision {
		b.WriteString("(none in this window)\n")
	}

	b.WriteString("\nALERTS SENT (time | kind | title | what it said)\n")
	for _, a := range in.Alerts {
		if time.Unix(a.TS, 0).Before(in.From) {
			continue
		}
		fmt.Fprintf(&b, "%s | %s | %s | %s\n", at(a.TS), a.Kind, a.Title, a.Body)
	}
	return b.String()
}
