package ford

import (
	"strings"
	"time"
)

// LastReported is the newest updateTime of any value in a flattened reading, in unix seconds, or 0 when the
// reading carries none. A parked car sends some values only when it wakes, so this is when the car (through
// Ford) last had something new to say, which is different from when CarTrak last asked Ford.
func LastReported(f map[string]any) int64 {
	var newest int64
	limit := time.Now().Add(24 * time.Hour).Unix() // ignore clock nonsense
	for k, v := range f {
		if !strings.HasSuffix(k, ".updateTime") {
			continue
		}
		str, ok := v.(string)
		if !ok {
			continue
		}
		if t, err := time.Parse(time.RFC3339, str); err == nil && t.Unix() > newest && t.Unix() < limit {
			newest = t.Unix()
		}
	}
	return newest
}
