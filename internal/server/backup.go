package server

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// backupInfo is what the Backup page shows.
// kvBackupTaken is when a backup was last downloaded, for the Getting started checklist.
const kvBackupTaken = "backup_taken_at" // the alerts package reads the same key for the backup reminder

func (s *Server) backupInfo(w http.ResponseWriter, r *http.Request) {
	ok(w, map[string]any{"dbBytes": s.S.DBSize(), "firstReading": s.S.FirstSampleTS()})
}

// backupDownload streams a consistent copy of the database. ?small=1 leaves out the raw Ford readings.
// The copy is made in the data folder and removed afterwards. Admin only.
func (s *Server) backupDownload(w http.ResponseWriter, r *http.Request) {
	small := r.URL.Query().Get("small") == "1"
	dir := s.DataDir
	if dir == "" {
		dir = os.TempDir()
	}
	tmp := filepath.Join(dir, fmt.Sprintf("backup-%d.tmp", time.Now().UnixNano()))
	defer os.Remove(tmp)
	if err := s.S.Backup(tmp, small); err != nil {
		fail(w, 500, "could not make the backup: "+err.Error())
		return
	}
	f, err := os.Open(tmp)
	if err != nil {
		fail(w, 500, "could not read the backup")
		return
	}
	defer f.Close()
	name := "cartrak-backup-" + time.Now().Format("2006-01-02")
	if small {
		name += "-small"
	}
	_ = s.S.SetKV(kvBackupTaken, strconv.FormatInt(time.Now().Unix(), 10))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`.db"`)
	w.Header().Set("Cache-Control", "no-store")
	if st, err := f.Stat(); err == nil {
		w.Header().Set("Content-Length", fmt.Sprint(st.Size()))
	}
	_, _ = io.Copy(w, f)
}
