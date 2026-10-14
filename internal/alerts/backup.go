package alerts

import (
	"strconv"
	"time"

	"github.com/ZachCurry13/cartrak/internal/push"
)

const (
	kvBackupTaken  = "backup_taken_at" // written when a backup is downloaded (see the server's backup handler)
	kvBackupAlert  = "alert_backup_at"
	backupEvery    = 30 * 24 * time.Hour
	backupMinUsage = 14 * 24 * time.Hour // not before CarTrak has been recording for two weeks
)

// backupCheck reminds, at most once a month, when there is data worth protecting and no backup in the last
// month. It never reminds a brand-new install.
func (e *Engine) backupCheck() {
	first := e.S.FirstTripTS()
	if first == 0 || e.now().Sub(time.Unix(first, 0)) < backupMinUsage {
		return
	}
	ago := func(key string) time.Duration {
		at, err := strconv.ParseInt(e.S.GetKV(key), 10, 64)
		if err != nil || at == 0 {
			return 1 << 62
		}
		return e.now().Sub(time.Unix(at, 0))
	}
	if ago(kvBackupTaken) < backupEvery || ago(kvBackupAlert) < backupEvery {
		return
	}
	_ = e.S.SetKV(kvBackupAlert, strconv.FormatInt(e.now().Unix(), 10))
	e.send(push.Alert{Kind: push.BackupReminder, Title: "Time for a backup", Body: "There is no backup from the last month. Download one under More, Backup so a failed disk cannot lose your history.", URL: "#/backup"})
}
