package ford

import (
	"regexp"
	"sort"
	"strings"
)

// These helpers let other packages read a flattened reading (State.Extra) by metric name,
// ignoring case and underscores, the same way the parser does.

// Find returns the value of the first metric found, trying each name in turn.
func Find(f map[string]any, names ...string) (any, bool) { return lookup(f, names...) }

// Text returns a metric as text ("" when absent).
func Text(f map[string]any, names ...string) string { return strAt(f, names...) }

// Number returns a metric as a number (nil when absent or not numeric).
func Number(f map[string]any, names ...string) *float64 { return numAt(f, names...) }

// ListItems returns the items of a list metric such as doorStatus, each as its fields in lower case.
func ListItems(f map[string]any, metric string) []map[string]any { return listItems(f, metric) }

// Updated is when Ford last updated one metric (its updateTime), or "". Different metrics
// update at different times: a parked car sends some of them only when it wakes up.
func Updated(f map[string]any, metric string) string { return strAt(f, metric+".updateTime") }

var indicatorKey = regexp.MustCompile(`(?i)(^|\.)indicators\.([^.]+)\.value$`)

// Indicators lists the warning lights that are on, by the name Ford gives them.
func Indicators(f map[string]any) []string {
	var on []string
	for k, v := range f {
		m := indicatorKey.FindStringSubmatch(k)
		if m == nil {
			continue
		}
		if b, ok := v.(bool); (ok && b) || strings.EqualFold(strAny(v), "ON") || strings.EqualFold(strAny(v), "TRUE") {
			on = append(on, m[2])
		}
	}
	sort.Strings(on)
	return on
}

func strAny(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
