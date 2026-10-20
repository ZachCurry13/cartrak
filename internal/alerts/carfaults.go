package alerts

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/push"
)

// The car runs its own checks and reports the result. These are the ones worth a message when they are anything but
// normal: the high-voltage battery's performance, the tire pressure system itself (not a low tire) and the 12 V load.
var carChecks = []struct {
	field, key, text string
	normal           func(string) bool // a value this check counts as fine besides the usual ones; nil when none
}{
	{"xevBatteryPerformanceStatus", "battery", "The car says the main battery is not performing normally (%s). Power or charging speed may be reduced.", nil},
	{"tirePressureSystemStatus", "tpms", "The tire pressure system reports %s. A sensor or the system itself may need attention.", nil},
	{"batteryLoadStatus", "load", "The car is managing the load on the 12 V battery (%s).", ford.LoadNormal},
}

// reportNormal says whether a status word means all is well, or that the car is not saying anything useful.
func reportNormal(v string) bool {
	switch strings.ToUpper(strings.TrimSpace(v)) {
	case "", "NORMAL", "NORMAL_OPERATION", "UNKNOWN", "UNSUPPORTED", "INVALID", "UNDEFINED", "NOT_AVAILABLE", "UNAVAILABLE", "NO_DATA":
		return true
	}
	return false
}

// carFaults alerts when the car reports something other than normal, at most once a day for each check. The time is
// kept so a restart does not repeat it.
func (e *Engine) carFaults(st *ford.State) {
	for _, c := range carChecks {
		v := ford.Text(st.Extra, c.field)
		if reportNormal(v) || (c.normal != nil && c.normal(v)) {
			continue
		}
		kv := "alert_fault_" + c.key
		if at, err := strconv.ParseInt(e.S.GetKV(kv), 10, 64); err == nil && e.now().Sub(time.Unix(at, 0)) < 24*time.Hour {
			continue
		}
		_ = e.S.SetKV(kv, strconv.FormatInt(e.now().Unix(), 10))
		e.send(push.Alert{Kind: push.CarFault, Title: "The car reports a problem", Body: fmt.Sprintf(c.text, strings.ToLower(strings.ReplaceAll(v, "_", " "))), URL: "#/health"})
	}
}
