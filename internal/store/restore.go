package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// RestoreDir is where an uploaded backup waits, and where a large backup can be copied by hand.
func RestoreDir(dataDir string) string { return filepath.Join(dataDir, "restore") }

// PendingName and ApplyFlag are the files inside the restore folder. The backup waits as pending.db; the apply
// flag says the owner confirmed, so the next start swaps it in.
const (
	PendingName = "pending.db"
	ApplyFlag   = "APPLY"
)

// BackupInfo is what a backup file holds, shown before the owner confirms a restore.
type BackupInfo struct {
	Bytes        int64    `json:"bytes"`
	Trips        int      `json:"trips"`
	Readings     int      `json:"readings"`
	Charges      int      `json:"charges"`
	FirstReading int64    `json:"firstReading"`
	LastReading  int64    `json:"lastReading"`
	Users        []string `json:"users"`
}

// InspectBackup checks that a file is a healthy CarTrak database and says what is in it. It never changes the file.
func InspectBackup(path string) (*BackupInfo, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	head := make([]byte, 16)
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	_, _ = f.Read(head)
	f.Close()
	if !strings.HasPrefix(string(head), "SQLite format 3") {
		return nil, fmt.Errorf("that is not a CarTrak backup (it is not a database file)")
	}
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path)+"?mode=ro&immutable=1")
	if err != nil {
		return nil, err
	}
	defer db.Close()
	var check string
	if err := db.QueryRow(`PRAGMA quick_check`).Scan(&check); err != nil || check != "ok" {
		return nil, fmt.Errorf("the file is damaged (%s)", check)
	}
	info := &BackupInfo{Bytes: st.Size(), Users: []string{}}
	for _, t := range []struct {
		table string
		into  *int
	}{{"trips", &info.Trips}, {"samples", &info.Readings}, {"charges", &info.Charges}} {
		if err := db.QueryRow(`SELECT COUNT(*) FROM ` + t.table).Scan(t.into); err != nil {
			return nil, fmt.Errorf("that is not a CarTrak backup (no %s table)", t.table)
		}
	}
	_ = db.QueryRow(`SELECT COALESCE(MIN(ts),0),COALESCE(MAX(ts),0) FROM samples`).Scan(&info.FirstReading, &info.LastReading)
	rows, err := db.Query(`SELECT name FROM users ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("that is not a CarTrak backup (no users table)")
	}
	defer rows.Close()
	for rows.Next() {
		var n string
		if rows.Scan(&n) == nil {
			info.Users = append(info.Users, n)
		}
	}
	if len(info.Users) == 0 {
		return nil, fmt.Errorf("that backup has no accounts, so nobody could sign in after restoring it")
	}
	return info, nil
}

// ApplyPendingRestore swaps in a confirmed backup. Call it at start-up, before the database is opened. The
// database being replaced is kept next to it as cartrak.db.before-restore-<time>. It reports whether a
// restore happened.
func ApplyPendingRestore(dataDir string) (bool, error) {
	dir := RestoreDir(dataDir)
	flag, pending := filepath.Join(dir, ApplyFlag), filepath.Join(dir, PendingName)
	if _, err := os.Stat(flag); err != nil {
		return false, nil
	}
	defer os.Remove(flag) // never apply twice
	if _, err := os.Stat(pending); err != nil {
		return false, nil
	}
	if _, err := InspectBackup(pending); err != nil {
		return false, fmt.Errorf("the backup was not restored: %w", err)
	}
	live := filepath.Join(dataDir, "cartrak.db")
	if _, err := os.Stat(live); err == nil {
		keep := fmt.Sprintf("%s.before-restore-%d", live, time.Now().Unix())
		if err := os.Rename(live, keep); err != nil {
			return false, fmt.Errorf("could not keep the current data safe, so nothing was restored: %w", err)
		}
	}
	_ = os.Remove(live + "-wal")
	_ = os.Remove(live + "-shm")
	if err := os.Rename(pending, live); err != nil {
		return false, fmt.Errorf("could not put the backup in place: %w", err)
	}
	return true, nil
}

// Checkpoint writes everything waiting in the journal into the main database file, so the file is complete
// when it is kept as a safety copy.
func (s *Store) Checkpoint() error {
	_, err := s.DB.Exec(`PRAGMA wal_checkpoint(TRUNCATE)`)
	return err
}
