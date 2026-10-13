package alerts

import (
	"fmt"
	"strings"

	"github.com/ZachCurry13/cartrak/internal/ford"
)

// plugged reads the plug state from either form Ford uses (CONNECTED, Disconnected). The second
// result is false when the car did not say.
func plugged(st *ford.State) (bool, bool) {
	p := strings.ToLower(st.Plug)
	switch {
	case p == "":
		return false, false
	case strings.Contains(p, "disconnect") || strings.Contains(p, "not"):
		return false, true
	case strings.Contains(p, "connect"):
		return true, true
	}
	return false, false
}

func charging(st *ford.State) bool {
	return (strings.Contains(strings.ToLower(st.ChargeState), "charging") && !strings.Contains(strings.ToLower(st.ChargeState), "not")) ||
		(st.ChargeKw != nil && *st.ChargeKw > 0.2)
}

// charge is " Battery at 82%." or "" when the car did not report a level.
func charge(st *ford.State) string {
	if st.SoCPct == nil {
		return ""
	}
	return fmt.Sprintf(" Battery at %.0f%%.", *st.SoCPct)
}

// wheel turns FRONT_LEFT into "Front left tire".
func wheel(w string) string {
	t := strings.ToLower(strings.ReplaceAll(w, "_", " "))
	if t == "" {
		return "A tire"
	}
	return strings.ToUpper(t[:1]) + t[1:] + " tire"
}
