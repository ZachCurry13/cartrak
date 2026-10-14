package alerts

import (
	"fmt"
	"strconv"
	"time"

	"github.com/ZachCurry13/cartrak/internal/diskfree"
	"github.com/ZachCurry13/cartrak/internal/push"
)

const kvDiskAlert = "alert_disk_at"

// diskCheck sends one alert when the disk holding the data is getting full, and then stays quiet for three days.
func (e *Engine) diskCheck() {
	if e.DiskSpace == nil {
		return
	}
	free, total, ok := e.DiskSpace()
	if !ok || diskfree.Level(free, total) == "ok" {
		return
	}
	if at, err := strconv.ParseInt(e.S.GetKV(kvDiskAlert), 10, 64); err == nil && e.now().Sub(time.Unix(at, 0)) < 72*time.Hour {
		return
	}
	_ = e.S.SetKV(kvDiskAlert, strconv.FormatInt(e.now().Unix(), 10))
	e.send(push.Alert{Kind: push.LowDisk, Title: "Low disk space", Body: fmt.Sprintf("%.1f GB left on the disk CarTrak uses. Free up space under More, Backup before it fills.", float64(free)/(1<<30)), URL: "#/backup"})
}
