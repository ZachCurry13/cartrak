package alerts

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/push"
)

const openAfter = 10 * time.Minute

// openThings lists what the car says is open: doors that are not closed, the hood, and any window that is down.
// Ford's list for the tailgate and the inner tailgate counts too.
func openThings(f map[string]any) []string {
	var out []string
	items := ford.ListItems(f, "doorStatus")
	passengerFront := false
	for _, it := range items {
		if d, _ := it["vehicledoor"].(string); strings.EqualFold(d, "UNSPECIFIED_FRONT") && strings.EqualFold(fmt.Sprint(it["vehicleside"]), "PASSENGER") {
			passengerFront = true
		}
	}
	for _, it := range items {
		v, _ := it["value"].(string)
		if v == "" || strings.EqualFold(v, "CLOSED") || strings.EqualFold(v, "UNKNOWN") {
			continue
		}
		door, _ := it["vehicledoor"].(string)
		side, _ := it["vehicleside"].(string)
		if strings.EqualFold(door, "UNSPECIFIED_FRONT") && !passengerFront {
			out = append(out, "front door (driver or passenger side)")
			continue
		}
		out = append(out, doorName(door, side))
	}
	if h := strings.ToUpper(ford.Text(f, "hoodStatus")); h != "" && h != "CLOSED" && h != "UNKNOWN" {
		out = append(out, "hood")
	}
	for k, v := range f {
		lk := strings.ToLower(k)
		if strings.Contains(lk, "windowstatus[") && strings.HasSuffix(lk, "doublerange.upperbound") {
			if n, ok := v.(float64); ok && n > 5 {
				out = append(out, "window")
				break
			}
		}
	}
	sort.Strings(out)
	return out
}

func doorName(door, side string) string {
	d := strings.ToLower(strings.ReplaceAll(door, "_", " "))
	switch d {
	case "unspecified front":
		if strings.EqualFold(side, "DRIVER") {
			return "driver door"
		}
		return "front passenger door"
	case "":
		return "a door"
	}
	return d + " door"
}

// leftOpen warns once when the car is parked with a door, the tailgate, the hood or a window open for ten minutes.
func (e *Engine) leftOpen(st *ford.State) {
	things := openThings(st.Extra)
	if len(things) == 0 || st.Driving() {
		e.openFrom, e.openSent = time.Time{}, false
		return
	}
	if e.openFrom.IsZero() {
		e.openFrom = st.TS
	}
	if !e.openSent && st.TS.Sub(e.openFrom) >= openAfter {
		e.openSent = true
		e.send(push.Alert{Kind: push.LeftOpen, Title: "Left open", Body: fmt.Sprintf("The parked car has its %s open.", strings.Join(things, ", ")), URL: "#/"})
	}
}
