package server

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

// A big database takes minutes to copy. One request that waits for the copy sends nothing meanwhile, and the Cloudflare tunnel
// gives up on a silent answer after about 100 seconds, so the Backup page asked for a file and never got one. The copy is made as
// a job instead: start it, ask how it is going, and download the finished file, which starts at once.

type backupJob struct {
	s       *Server
	small   bool
	started time.Time
	state   string // working, ready, failed
	err     string
	path    string
	size    int64
}

var (
	jobsMu sync.Mutex
	jobs   = map[string]*backupJob{}
)

func (s *Server) backupStart(w http.ResponseWriter, r *http.Request) {
	small := r.URL.Query().Get("small") == "1"
	dir := s.DataDir
	if dir == "" {
		dir = os.TempDir()
	}
	jobsMu.Lock()
	for id, j := range jobs { // old jobs and their files go after an hour
		if time.Since(j.started) > time.Hour {
			if j.path != "" {
				_ = os.Remove(j.path)
			}
			delete(jobs, id)
		}
	}
	for _, j := range jobs {
		if j.s == s && j.state == "working" {
			jobsMu.Unlock()
			fail(w, 409, "a backup is already being made, wait for it to finish")
			return
		}
	}
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	id := hex.EncodeToString(b)
	j := &backupJob{s: s, small: small, started: time.Now(), state: "working"}
	jobs[id] = j
	jobsMu.Unlock()
	go func() {
		tmp := filepath.Join(dir, fmt.Sprintf("backup-%d.tmp", time.Now().UnixNano()))
		backup := s.S.Backup
		if s.Core != nil {
			backup = s.S.BackupCar
		}
		err := backup(tmp, small)
		jobsMu.Lock()
		defer jobsMu.Unlock()
		if err != nil {
			_ = os.Remove(tmp)
			j.state, j.err = "failed", "could not make the backup: "+err.Error()
			return
		}
		if st, e := os.Stat(tmp); e == nil {
			j.size = st.Size()
		}
		j.path, j.state = tmp, "ready"
	}()
	ok(w, map[string]string{"id": id})
}

func (s *Server) backupJobOf(id string) *backupJob {
	jobsMu.Lock()
	defer jobsMu.Unlock()
	if j := jobs[id]; j != nil && j.s == s {
		return j
	}
	return nil
}

func (s *Server) backupStatus(w http.ResponseWriter, r *http.Request) {
	j := s.backupJobOf(r.URL.Query().Get("id"))
	if j == nil {
		fail(w, 404, "that backup is gone, start it again")
		return
	}
	jobsMu.Lock()
	defer jobsMu.Unlock()
	ok(w, map[string]any{"state": j.state, "error": j.err, "bytes": j.size, "seconds": int(time.Since(j.started).Seconds())})
}

// backupFile hands over a finished copy once, and removes it afterwards.
func (s *Server) backupFile(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	j := s.backupJobOf(id)
	jobsMu.Lock()
	ready := j != nil && j.state == "ready"
	var path string
	var small bool
	if ready {
		path, small = j.path, j.small
	}
	jobsMu.Unlock()
	if !ready {
		fail(w, 404, "that backup is not ready or is gone, start it again")
		return
	}
	f, err := os.Open(path)
	if err != nil {
		fail(w, 500, "could not read the backup")
		return
	}
	defer f.Close()
	name := "cartrak-backup-" + time.Now().Format("2006-01-02")
	if s.CarID > 1 {
		name += "-car" + strconv.FormatInt(s.CarID, 10)
	}
	if small {
		name += "-small"
	}
	_ = s.S.SetKV(kvBackupTaken, strconv.FormatInt(time.Now().Unix(), 10))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`.db"`)
	w.Header().Set("Cache-Control", "no-store")
	if st, e := f.Stat(); e == nil {
		w.Header().Set("Content-Length", fmt.Sprint(st.Size()))
	}
	_, _ = io.Copy(w, f)
	// kept until it has been sent in full; a second download of the same job starts a new copy
	jobsMu.Lock()
	delete(jobs, id)
	jobsMu.Unlock()
	_ = f.Close()
	_ = os.Remove(path)
}
