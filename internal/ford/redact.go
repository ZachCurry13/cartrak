package ford

import (
	"encoding/json"
	"regexp"
	"strings"
	"unicode"
)

// Redact removes location and vehicle-identity values from a JSON document so
// it can be shared for debugging. Keys and structure are kept; only the values
// of sensitive fields (latitude, longitude, position, VIN, address...) change.
func Redact(raw []byte) any {
	var v any
	if json.Unmarshal(raw, &v) != nil {
		return string(truncate(raw, 2000))
	}
	return redact(v, false)
}

func truncate(b []byte, n int) []byte {
	if len(b) > n {
		return b[:n]
	}
	return b
}

var sensitiveTokens = map[string]bool{"lat": true, "latitude": true, "lon": true, "lng": true, "long": true, "longitude": true,
	"vin": true, "position": true, "address": true, "coordinate": true, "coordinates": true, "gnss": true, "gps": true,
	"location": true, "heading": true, "plate": true}

var vinLike = regexp.MustCompile(`^[A-HJ-NPR-Z0-9]{17}$`)

// tokens splits camelCase, snake_case and dotted names into lower-case words.
func tokens(s string) []string {
	var out []string
	var cur []rune
	flush := func() {
		if len(cur) > 0 {
			out = append(out, strings.ToLower(string(cur)))
			cur = cur[:0]
		}
	}
	rs := []rune(s)
	for i, r := range rs {
		switch {
		case !unicode.IsLetter(r) && !unicode.IsDigit(r):
			flush()
		case unicode.IsUpper(r) && i > 0 && (unicode.IsLower(rs[i-1]) || (i+1 < len(rs) && unicode.IsLower(rs[i+1]) && unicode.IsUpper(rs[i-1]))):
			flush()
			cur = append(cur, r)
		default:
			cur = append(cur, r)
		}
	}
	flush()
	return out
}

func sensitive(name string) bool {
	for _, t := range tokens(name) {
		if sensitiveTokens[t] {
			return true
		}
	}
	return false
}

func redact(v any, hide bool) any {
	switch t := v.(type) {
	case map[string]any:
		// A metric object like {"name":"POSITION", "value":{...}}: hide its payload.
		for _, nk := range nameKeys {
			if s, ok := t[nk].(string); ok && sensitive(s) {
				hide = true
			}
		}
		out := make(map[string]any, len(t))
		for k, c := range t {
			isName := false
			for _, nk := range nameKeys {
				if k == nk {
					isName = true
				}
			}
			out[k] = redact(c, !isName && (hide || sensitive(k)))
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, c := range t {
			out[i] = redact(c, hide)
		}
		return out
	case string:
		if hide || vinLike.MatchString(t) {
			return "<redacted>"
		}
	case float64, bool:
		if hide {
			return "<redacted>"
		}
	}
	return v
}
