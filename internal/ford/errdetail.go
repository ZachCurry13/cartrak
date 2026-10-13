package ford

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
)

var vinInText = regexp.MustCompile(`[A-HJ-NPR-Z0-9]{17}`)

// errorDetail is a short, safe summary of the body Ford sent with an error: its message fields
// if it is JSON, otherwise the first part of the text. VIN-like text is removed and the answer is
// capped, so it can be shown on the Status screen and written to the log.
func errorDetail(b []byte) string {
	t := strings.TrimSpace(string(b))
	if t == "" {
		return ""
	}
	var v map[string]any
	if json.Unmarshal(b, &v) == nil {
		var parts []string
		for _, k := range []string{"message", "error", "error_description", "errorMessage", "description", "status", "code"} {
			switch x := v[k].(type) {
			case string:
				if x != "" {
					parts = append(parts, k+": "+x)
				}
			case float64:
				parts = append(parts, k+": "+strconv.FormatFloat(x, 'f', -1, 64))
			}
		}
		if len(parts) > 0 {
			t = strings.Join(parts, ", ")
		}
	}
	t = strings.Join(strings.Fields(t), " ")
	t = vinInText.ReplaceAllString(t, "[vin]")
	if len(t) > 200 {
		t = t[:200] + "..."
	}
	return t
}
