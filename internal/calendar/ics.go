// Package calendar reads a calendar's private ICS address (Google's "secret address in iCal format", Outlook's published
// calendar) and lists the appointments of a day, so the route planner can fill them in and trips can be named after them.
package calendar

import (
	"strings"
	"time"
)

// Event is one VEVENT as written: a single appointment, or the first of a repeating one.
type Event struct {
	UID, Title, Location string
	Start, End           time.Time
	AllDay               bool
	Rule                 string      // the RRULE, "" for a single appointment
	Except               []time.Time // EXDATE: occurrences taken out
	RecurrenceID         *time.Time  // set on a changed occurrence of a repeating appointment: the one it replaces
	Cancelled            bool
}

// Parse reads the events of an ICS file. Lines it does not understand are skipped; an event without a start is dropped.
// local is the zone for times written without one.
func Parse(text string, local *time.Location) []Event {
	var out []Event
	var cur *Event
	var hasEnd bool
	var dur time.Duration
	for _, line := range unfold(text) {
		name, params, value := split(line)
		switch {
		case name == "BEGIN" && value == "VEVENT":
			cur, hasEnd, dur = &Event{}, false, 0
		case name == "END" && value == "VEVENT" && cur != nil:
			if !cur.Start.IsZero() {
				if !hasEnd {
					cur.End = cur.Start.Add(dur)
					if dur == 0 && cur.AllDay {
						cur.End = cur.Start.AddDate(0, 0, 1)
					}
				}
				out = append(out, *cur)
			}
			cur = nil
		case cur == nil:
		case name == "UID":
			cur.UID = value
		case name == "SUMMARY":
			cur.Title = unescape(value)
		case name == "LOCATION":
			cur.Location = unescape(value)
		case name == "STATUS":
			cur.Cancelled = strings.EqualFold(value, "CANCELLED")
		case name == "RRULE":
			cur.Rule = value
		case name == "DTSTART":
			cur.Start, cur.AllDay = parseTime(value, params, local)
		case name == "DTEND":
			cur.End, _ = parseTime(value, params, local)
			hasEnd = !cur.End.IsZero()
		case name == "DURATION":
			dur = parseDuration(value)
		case name == "EXDATE":
			for _, v := range strings.Split(value, ",") {
				if t, _ := parseTime(v, params, local); !t.IsZero() {
					cur.Except = append(cur.Except, t)
				}
			}
		case name == "RECURRENCE-ID":
			if t, _ := parseTime(value, params, local); !t.IsZero() {
				cur.RecurrenceID = &t
			}
		}
	}
	return out
}

// unfold joins the continuation lines of an ICS file (a line starting with a space or a tab continues the one before).
func unfold(text string) []string {
	var out []string
	for _, l := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		if (strings.HasPrefix(l, " ") || strings.HasPrefix(l, "\t")) && len(out) > 0 {
			out[len(out)-1] += l[1:]
			continue
		}
		out = append(out, strings.TrimRight(l, "\r"))
	}
	return out
}

// split cuts "DTSTART;TZID=America/New_York:20261113T090000" into its name, parameters and value.
func split(line string) (string, map[string]string, string) {
	head, value, ok := strings.Cut(line, ":")
	if !ok {
		return "", nil, ""
	}
	parts := strings.Split(head, ";")
	params := map[string]string{}
	for _, p := range parts[1:] {
		if k, v, ok := strings.Cut(p, "="); ok {
			params[strings.ToUpper(k)] = strings.Trim(v, `"`)
		}
	}
	return strings.ToUpper(parts[0]), params, strings.TrimSpace(value)
}

func unescape(s string) string {
	return strings.NewReplacer(`\n`, ", ", `\N`, ", ", `\,`, ",", `\;`, ";", `\\`, `\`).Replace(s)
}

func parseTime(v string, params map[string]string, local *time.Location) (time.Time, bool) {
	v = strings.TrimSpace(v)
	if params["VALUE"] == "DATE" || len(v) == 8 {
		t, err := time.ParseInLocation("20060102", v, local)
		if err != nil {
			return time.Time{}, false
		}
		return t, true
	}
	if strings.HasSuffix(v, "Z") {
		t, _ := time.Parse("20060102T150405Z", v)
		return t, false
	}
	loc := local
	if tz := params["TZID"]; tz != "" {
		loc = zone(tz, local)
	}
	t, _ := time.ParseInLocation("20060102T150405", v, loc)
	return t, false
}

// windowsZones are the zone names Outlook writes, for the US ones.
var windowsZones = map[string]string{
	"Eastern Standard Time": "America/New_York", "Central Standard Time": "America/Chicago", "Mountain Standard Time": "America/Denver",
	"US Mountain Standard Time": "America/Phoenix", "Pacific Standard Time": "America/Los_Angeles", "Alaskan Standard Time": "America/Anchorage",
	"Hawaiian Standard Time": "Pacific/Honolulu", "UTC": "UTC",
}

func zone(name string, fallback *time.Location) *time.Location {
	if w, ok := windowsZones[name]; ok {
		name = w
	}
	if l, err := time.LoadLocation(name); err == nil {
		return l
	}
	return fallback
}

// parseDuration reads the common ICS durations: PT1H30M, PT45M, P1D.
func parseDuration(v string) time.Duration {
	var d time.Duration
	n := 0
	for _, r := range strings.TrimPrefix(strings.TrimPrefix(v, "+"), "P") {
		switch {
		case r >= '0' && r <= '9':
			n = n*10 + int(r-'0')
		case r == 'W':
			d += time.Duration(n) * 7 * 24 * time.Hour
			n = 0
		case r == 'D':
			d += time.Duration(n) * 24 * time.Hour
			n = 0
		case r == 'H':
			d += time.Duration(n) * time.Hour
			n = 0
		case r == 'M':
			d += time.Duration(n) * time.Minute
			n = 0
		case r == 'S':
			d += time.Duration(n) * time.Second
			n = 0
		}
	}
	return d
}
