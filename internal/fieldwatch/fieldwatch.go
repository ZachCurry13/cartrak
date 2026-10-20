// Package fieldwatch looks across many stored readings and says, for each thing the car reports, how often it changes,
// when it last changed and what it has been: so nothing the car sends goes unnoticed.
package fieldwatch

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Row is one stored reading, flattened to dotted-path -> value (the form ford.Flatten gives).
type Row struct {
	TS   int64
	Flat map[string]any
}

// Stat is what one metric did across the rows.
type Stat struct {
	Metric       string   `json:"metric"`
	Samples      int      `json:"samples"`      // readings that carried it
	Changes      int      `json:"changes"`      // times its value differed from the reading before
	Last         string   `json:"last"`         // its latest value, in short
	LastChangeTS int64    `json:"lastChangeTs"` // when it was last seen to change; 0 when it never did
	ReportedAt   string   `json:"reportedAt"`   // the car's own time for its latest value
	Numeric      bool     `json:"numeric"`
	Min          *float64 `json:"min"`
	Max          *float64 `json:"max"`
	Parts        int      `json:"parts"` // separate values it is made of (a tire list has four)
}

const prefix = "telemetry.metrics."

// metricOf is the top-level name: "telemetry.metrics.doorStatus[3].value" is "doorStatus".
func metricOf(key string) string {
	k := strings.TrimPrefix(key, prefix)
	if strings.HasPrefix(k, "garage.") || strings.HasPrefix(k, "telemetry.") || strings.HasPrefix(k, "info.") {
		return ""
	}
	if i := strings.IndexAny(k, ".["); i > 0 {
		k = k[:i]
	}
	if k == "" || k == strings.ToUpper(k) { // ALL_CAPS names are aliases for the same thing
		return ""
	}
	return k
}

// isValue says whether a path is a value worth comparing: not a timestamp, an id or a label that rides along.
func isValue(rest string) bool {
	if strings.HasSuffix(rest, ".updateTime") || strings.Contains(rest, "oemCorrelationId") || strings.Contains(rest, ".tags.") {
		return false
	}
	return strings.Contains(rest, ".value") || strings.HasSuffix(rest, "value")
}

func short(v any) string {
	switch t := v.(type) {
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case string:
		return t
	}
	return fmt.Sprint(v)
}

type acc struct {
	Stat
	sig   string
	vals  []string
	start bool
}

// Analyze goes through rows (any order) and returns one Stat per metric, busiest first.
func Analyze(rows []Row) []Stat {
	sorted := append([]Row(nil), rows...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].TS < sorted[j].TS })
	by := map[string]*acc{}
	for _, r := range sorted {
		type part struct{ k, v string }
		parts := map[string][]part{}
		times := map[string]string{}
		nums := map[string]float64{}
		for k, v := range r.Flat {
			m := metricOf(k)
			if m == "" {
				continue
			}
			rest := strings.TrimPrefix(k, prefix)[len(m):]
			if strings.HasSuffix(rest, ".updateTime") && !strings.Contains(rest, "[") {
				if s, ok := v.(string); ok && s > times[m] {
					times[m] = s
				}
				continue
			}
			if !isValue(rest) {
				continue
			}
			parts[m] = append(parts[m], part{rest, short(v)})
			if f, ok := v.(float64); ok && (rest == ".value") {
				nums[m] = f
			}
		}
		for m, ps := range parts {
			sort.Slice(ps, func(i, j int) bool { return ps[i].k < ps[j].k })
			var sig strings.Builder
			vals := make([]string, 0, len(ps))
			for _, p := range ps {
				sig.WriteString(p.k + "=" + p.v + ";")
				vals = append(vals, p.v)
			}
			a := by[m]
			if a == nil {
				a = &acc{Stat: Stat{Metric: m}}
				by[m] = a
			}
			a.Samples++
			a.Parts = len(ps)
			if a.start && a.sig != sig.String() {
				a.Changes++
				a.LastChangeTS = r.TS
			}
			a.start, a.sig, a.vals = true, sig.String(), vals
			if t, ok := times[m]; ok {
				a.ReportedAt = t
			}
			if f, ok := nums[m]; ok {
				a.Numeric = true
				if a.Min == nil || f < *a.Min {
					v := f
					a.Min = &v
				}
				if a.Max == nil || f > *a.Max {
					v := f
					a.Max = &v
				}
			}
		}
	}
	out := make([]Stat, 0, len(by))
	for _, a := range by {
		n := len(a.vals)
		if n > 4 {
			n = 4
		}
		a.Last = strings.Join(a.vals[:n], " / ")
		if len(a.vals) > 4 {
			a.Last += " …"
		}
		if private(a.Metric) {
			a.Last, a.Min, a.Max, a.ReportedAt = "hidden: a position or an identifier", nil, nil, ""
		}
		out = append(out, a.Stat)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Changes != out[j].Changes {
			return out[i].Changes > out[j].Changes
		}
		return out[i].Metric < out[j].Metric
	})
	return out
}

// Age is a short "3 h ago" for the page's help text in tests and logs.
func Age(ts, now int64) string {
	d := time.Duration(now-ts) * time.Second
	switch {
	case ts == 0:
		return "never"
	case d < time.Hour:
		return fmt.Sprintf("%d min ago", int(d.Minutes()))
	case d < 48*time.Hour:
		return fmt.Sprintf("%d h ago", int(d.Hours()))
	}
	return fmt.Sprintf("%d d ago", int(d.Hours()/24))
}

// private says whether a metric holds where the car is or who it is. Its activity is listed, its value never is: the
// Car data page makes the same promise, and this page may be seen by viewers.
func private(metric string) bool {
	m := strings.ToLower(metric)
	for _, w := range []string{"position", "location", "latitude", "longitude", "gps", "vin", "vehicleid", "geo", "address"} {
		if strings.Contains(m, w) {
			return true
		}
	}
	return false
}
