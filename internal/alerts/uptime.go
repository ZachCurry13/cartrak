package alerts

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ZachCurry13/cartrak/internal/push"
)

const (
	kvAlive       = "alive_ts"       // the last time CarTrak was running, kept up to date every minute
	kvHeartbeat   = "heartbeat_last" // how the last ping of the uptime monitor went
	downAfter     = 5 * time.Minute  // a gap this long between runs counts as having been offline (a restart to update is shorter)
	heartbeatEach = 5 * time.Minute
)

// HeartbeatResult is how the last ping of the outside uptime monitor went.
type HeartbeatResult struct {
	TS  int64  `json:"ts"`
	OK  bool   `json:"ok"`
	Msg string `json:"msg"`
}

// upCheck runs when CarTrak starts: if it was last running more than a few minutes ago, it was offline, and it says so now that it
// is back. It cannot say so while it is down; for that an outside monitor is needed (the heartbeat below).
func (e *Engine) upCheck() {
	e.mu.Lock()
	defer e.mu.Unlock()
	now := e.now()
	var prev int64
	fmt.Sscan(e.S.GetKV(kvAlive), &prev)
	_ = e.S.SetKV(kvAlive, fmt.Sprint(now.Unix()))
	gap := now.Sub(time.Unix(prev, 0))
	if prev == 0 || gap < downAfter {
		return
	}
	e.send(push.Alert{Kind: push.ServerBack, Title: "CarTrak is back online",
		Body: fmt.Sprintf("It was offline for about %s (from %s). Readings and trips during that time may be missing.", span(gap), time.Unix(prev, 0).In(e.loc()).Format("Jan 2, 3:04 PM")), URL: "#/status"})
}

func span(d time.Duration) string {
	m := int(d.Minutes() + 0.5)
	if m < 120 {
		return fmt.Sprintf("%d minutes", m)
	}
	return fmt.Sprintf("%.1f hours", d.Hours())
}

func (e *Engine) loc() *time.Location {
	if e.Loc != nil {
		if l := e.Loc(); l != nil {
			return l
		}
	}
	return time.Local
}

// beat runs every minute: it notes that CarTrak is running and, when an uptime monitor address is set, pings it every five minutes.
// A monitor that stops hearing from CarTrak tells you it is down, and tells you again when the pings come back.
func (e *Engine) beat() {
	e.mu.Lock()
	now := e.now()
	_ = e.S.SetKV(kvAlive, fmt.Sprint(now.Unix()))
	var url string
	if e.HeartbeatURL != nil {
		url = e.HeartbeatURL()
	}
	due := url != "" && now.Sub(e.lastPing) >= heartbeatEach
	if due {
		e.lastPing = now
	}
	e.mu.Unlock()
	if !due {
		return
	}
	err := PingURL(context.Background(), e.HTTP, url)
	res := HeartbeatResult{TS: now.Unix(), OK: err == nil, Msg: "It answered."}
	if err != nil {
		res.Msg = err.Error()
	}
	b, _ := json.Marshal(res)
	_ = e.S.SetKV(kvHeartbeat, string(b))
}

// PingURL asks an address once, with a short wait, and says whether it answered with success.
func PingURL(ctx context.Context, hc *http.Client, url string) error {
	if hc == nil {
		hc = &http.Client{Timeout: 10 * time.Second}
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "CarTrak heartbeat")
	resp, err := hc.Do(req)
	if err != nil {
		return fmt.Errorf("the monitor did not answer")
	}
	resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("the monitor answered %d", resp.StatusCode)
	}
	return nil
}
