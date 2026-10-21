package alerts

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/push"
)

const (
	openAfter  = 10 * time.Minute
	openRepeat = 12 * time.Hour // the same things open are told about at most this often
	kvOpen     = "alert_open_last"
)

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

// openStamp is when Ford says the car last reported its doors ("" when unknown). A parked car stops reporting, so a door shown
// open can be the last thing the car said before it went to sleep.
func openStamp(f map[string]any) string {
	latest := ""
	for _, it := range ford.ListItems(f, "doorStatus") {
		if at, _ := it["updatetime"].(string); at > latest { // RFC 3339 times sort as text
			latest = at
		}
	}
	return latest
}

// lockStamp is when Ford says the car last reported its locks ("" when unknown).
func lockStamp(f map[string]any) string {
	latest := ""
	for _, it := range ford.ListItems(f, "doorLockStatus") {
		if at, _ := it["updatetime"].(string); at > latest {
			latest = at
		}
	}
	return latest
}

// reportedAt says when the car last reported, in the local time zone.
func (e *Engine) reportedAt(stamp string) string {
	t, err := time.Parse(time.RFC3339, stamp)
	if err != nil {
		return ""
	}
	t = t.In(e.zone())
	if t.YearDay() == e.now().In(e.zone()).YearDay() && t.Year() == e.now().In(e.zone()).Year() {
		return t.Format("3:04 pm")
	}
	return t.Format("Mon 3:04 pm")
}

// leftOpen warns once when the car is parked with a door, the tailgate, the hood or a window open for ten minutes.
func (e *Engine) leftOpen(st *ford.State) {
	things := openThings(st.Extra)
	if len(things) == 0 || st.Driving() {
		e.openFrom, e.openSent = time.Time{}, false
		if len(things) == 0 && len(ford.ListItems(st.Extra, "doorStatus")) > 0 {
			// a report of everything closed that is newer than the one alerted about: the next opening is news
			if _, rest, ok := strings.Cut(e.S.GetKV(kvOpen), "|"); ok {
				if _, was, _ := strings.Cut(rest, "|"); was == "" || openStamp(st.Extra) == "" || openStamp(st.Extra) > was {
					_ = e.S.SetKV(kvOpen, "")
				}
			}
		}
		return
	}
	if e.openFrom.IsZero() {
		e.openFrom = st.TS
	}
	stamp := openStamp(st.Extra)
	key := strings.Join(things, ",")
	if !e.openSent && st.TS.Sub(e.openFrom) >= openAfter {
		e.openSent = true
		// The same things open: at most one alert in openRepeat, kept across restarts. A sleeping car repeats its last report
		// (and Ford's list of doors sometimes comes without them), which would otherwise alert again and again.
		if k, rest, ok := strings.Cut(e.S.GetKV(kvOpen), "|"); ok && k == key {
			at, _, _ := strings.Cut(rest, "|")
			if n, err := strconv.ParseInt(at, 10, 64); err == nil && st.TS.Sub(time.Unix(n, 0)) < openRepeat {
				return
			}
		}
		_ = e.S.SetKV(kvOpen, key+"|"+strconv.FormatInt(st.TS.Unix(), 10)+"|"+stamp) // what, when, and the car's report time
		body := fmt.Sprintf("The parked car has its %s open.", strings.Join(things, ", "))
		if at := e.reportedAt(stamp); at != "" {
			body += fmt.Sprintf(" The car last reported this at %s, so it may have been closed since.", at)
		}
		e.send(push.Alert{Kind: push.LeftOpen, Title: "Left open", Body: body, URL: "#/"})
	}
}
