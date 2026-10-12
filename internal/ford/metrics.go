package ford

import (
	"regexp"
	"strings"
)

var metricName = regexp.MustCompile(`^[A-Z][A-Z0-9_]{3,}$`)

// nameKeys are field names that may hold a metric's name in a list of metrics.
var nameKeys = []string{"name", "metricName", "metric", "id", "key"}

// valueKeys are typed value fields (the data dictionary lists DOUBLE_VALUE, ENUM_VALUE...).
var valueKeys = map[string]bool{"doubleValue": true, "double_value": true, "stringValue": true, "string_value": true,
	"enumValue": true, "enum_value": true, "int64Value": true, "int64_value": true, "intValue": true, "boolValue": true, "bool_value": true}

// addMetricAliases handles responses that list metrics as objects, for example
// {"name":"XEV_BATTERY_STATE_OF_CHARGE","value":81.2} inside an array. It adds
// NAME.<field> (and NAME.value for typed value fields) so a lookup by the
// dictionary's metric name works wherever the metric sits in the document.
func addMetricAliases(f map[string]any) {
	type pair struct{ prefix, name string }
	var pairs []pair
	for p, v := range f {
		s, ok := v.(string)
		if !ok || !metricName.MatchString(s) {
			continue
		}
		for _, nk := range nameKeys {
			if strings.HasSuffix(p, "."+nk) {
				pairs = append(pairs, pair{strings.TrimSuffix(p, "."+nk), s})
				break
			}
		}
	}
	if len(pairs) == 0 {
		return
	}
	add := map[string]any{}
	for _, pr := range pairs {
		pre := pr.prefix + "."
		for q, v := range f {
			if !strings.HasPrefix(q, pre) {
				continue
			}
			rest := q[len(pre):]
			add[pr.name+"."+rest] = v
			if first := strings.SplitN(rest, ".", 2); valueKeys[first[0]] {
				add[pr.name+".value"] = v
			}
		}
	}
	for k, v := range add {
		if _, exists := f[k]; !exists {
			f[k] = v
		}
	}
}
