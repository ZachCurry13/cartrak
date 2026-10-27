package alerts

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/push"
)

func (e *Engine) unlocked(st *ford.State) {
	if st.Locked != nil && *st.Locked {
		_ = e.S.SetKV(kvUnlocked, "") // locked again: the next time it is left unlocked is news
	}
	if st.Locked == nil || *st.Locked || st.Driving() { // ignition on means someone is in the car, even while it charges
		e.unlockedFrom, e.unlockedSent = time.Time{}, false
		return
	}
	if e.unlockedFrom.IsZero() {
		e.unlockedFrom = st.TS
	}
	if !e.unlockedSent && st.TS.Sub(e.unlockedFrom) >= unlockedAfter {
		e.unlockedSent = true
		stamp := lockStamp(st.Extra)
		// The same report is announced once, kept across restarts: a sleeping car repeats its last report for days, and every update
		// restarts CarTrak, which would otherwise announce it again each time.
		if prev := e.S.GetKV(kvUnlocked); prev != "" {
			was, at, _ := strings.Cut(prev, "|")
			n, _ := strconv.ParseInt(at, 10, 64)
			if (stamp != "" && was == stamp) || (stamp == "" && e.now().Sub(time.Unix(n, 0)) < unlockedAgain) {
				return
			}
		}
		_ = e.S.SetKV(kvUnlocked, stamp+"|"+strconv.FormatInt(e.now().Unix(), 10))
		body := "Parked and unlocked for over 10 minutes."
		if stamp != "" {
			if t, err := time.Parse(time.RFC3339, stamp); err == nil && st.TS.Sub(t) > 30*time.Minute { // a sleeping car repeats its last report
				body = fmt.Sprintf("The car says it is unlocked. It last reported this at %s, so it may have been locked since.", e.reportedAt(stamp))
			}
		}
		e.send(push.Alert{Kind: push.Unlocked, Title: "Car left unlocked", Body: body, URL: "#/"})
	}
}
