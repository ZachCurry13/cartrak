package alerts

import (
	"fmt"
	"strconv"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/push"
)

const (
	kvSilentAlert = "alert_silent_at"
	silentAfter   = 24 * time.Hour // a parked car sleeps for hours; a day with nothing new is something else
)

// carSilent sends one alert, at most once a day, when the car's own newest report is more than a day old. A weak 12 V battery is
// the usual reason a connected car stops reporting; it can also be a car parked without cell signal.
func (e *Engine) carSilent(st *ford.State) {
	last := ford.LastReported(st.Extra)
	if last == 0 || st.TS.Sub(time.Unix(last, 0)) < silentAfter { // against the reading's own time: that is when CarTrak asked
		return
	}
	if at, err := strconv.ParseInt(e.S.GetKV(kvSilentAlert), 10, 64); err == nil && e.now().Sub(time.Unix(at, 0)) < 24*time.Hour {
		return
	}
	_ = e.S.SetKV(kvSilentAlert, strconv.FormatInt(e.now().Unix(), 10))
	when := e.reportedWhen(time.Unix(last, 0).UTC().Format(time.RFC3339))
	e.send(push.Alert{Kind: push.CarSilent, Title: "Car stopped reporting", Body: fmt.Sprintf("The car last reported %s. A parked car sleeps, but this long can mean a weak 12 V battery or no cell signal.", when), URL: "#/health"})
}
