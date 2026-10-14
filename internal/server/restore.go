package server

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

const maxRestoreBytes = 200 << 30

// restoreInfo says what is waiting to be restored: an uploaded backup, and any .db files copied into the
// restore folder by hand (for backups too big to upload).
func (s *Server) restoreInfo(w http.ResponseWriter, r *http.Request) {
	dir := store.RestoreDir(s.DataDir)
	out := map[string]any{"folder": "restore", "files": []string{}}
	if info, err := store.InspectBackup(filepath.Join(dir, store.PendingName)); err == nil {
		out["pending"] = info
	}
	if entries, err := os.ReadDir(dir); err == nil {
		var names []string
		for _, e := range entries {
			if n := e.Name(); !e.IsDir() && strings.HasSuffix(n, ".db") && n != store.PendingName {
				names = append(names, n)
			}
		}
		if names != nil {
			out["files"] = names
		}
	}
	ok(w, out)
}

// restoreUpload receives a backup (the file as the request body), checks it, and keeps it waiting. Nothing is
// replaced yet. Admin only.
func (s *Server) restoreUpload(w http.ResponseWriter, r *http.Request) {
	dir := store.RestoreDir(s.DataDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fail(w, 500, "could not create the restore folder")
		return
	}
	tmp := filepath.Join(dir, "upload.tmp")
	f, err := os.Create(tmp)
	if err != nil {
		fail(w, 500, "could not save the upload")
		return
	}
	n, err := io.Copy(f, http.MaxBytesReader(w, r.Body, maxRestoreBytes))
	f.Close()
	if err != nil || n == 0 {
		os.Remove(tmp)
		fail(w, 400, "the upload did not finish")
		return
	}
	s.stagePending(w, tmp)
}

// restoreUseFile checks a .db file the owner copied into the restore folder. Admin only.
func (s *Server) restoreUseFile(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name string `json:"name"`
	}
	dir := store.RestoreDir(s.DataDir)
	if decode(r, &in) != nil || in.Name == "" || in.Name != filepath.Base(in.Name) || !strings.HasSuffix(in.Name, ".db") || in.Name == store.PendingName {
		fail(w, 400, "choose a .db file from the restore folder")
		return
	}
	src := filepath.Join(dir, in.Name)
	tmp := filepath.Join(dir, "upload.tmp")
	if err := os.Rename(src, tmp); err != nil {
		fail(w, 404, "that file is not in the restore folder")
		return
	}
	s.stagePending(w, tmp)
}

func (s *Server) stagePending(w http.ResponseWriter, tmp string) {
	dir := filepath.Dir(tmp)
	info, err := store.InspectBackup(tmp)
	if err != nil {
		os.Remove(tmp)
		fail(w, 400, err.Error())
		return
	}
	_ = os.Remove(filepath.Join(dir, store.ApplyFlag))
	if err := os.Rename(tmp, filepath.Join(dir, store.PendingName)); err != nil {
		fail(w, 500, "could not keep the backup")
		return
	}
	ok(w, info)
}

// restoreDiscard throws away the waiting backup. Admin only.
func (s *Server) restoreDiscard(w http.ResponseWriter, r *http.Request) {
	dir := store.RestoreDir(s.DataDir)
	_ = os.Remove(filepath.Join(dir, store.ApplyFlag))
	_ = os.Remove(filepath.Join(dir, store.PendingName))
	ok(w, "discarded")
}

// restoreApply confirms the restore and stops CarTrak. On the next start the backup is swapped in and the
// current data is kept beside it. It needs the owner to type RESTORE. Admin only.
func (s *Server) restoreApply(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Confirm string `json:"confirm"`
	}
	if decode(r, &in) != nil || in.Confirm != "RESTORE" {
		fail(w, 400, "type RESTORE to confirm")
		return
	}
	dir := store.RestoreDir(s.DataDir)
	if _, err := store.InspectBackup(filepath.Join(dir, store.PendingName)); err != nil {
		fail(w, 400, "there is no valid backup waiting: "+err.Error())
		return
	}
	if err := s.S.Checkpoint(); err != nil {
		fail(w, 500, "could not prepare the current data to be kept safe")
		return
	}
	if err := os.WriteFile(filepath.Join(dir, store.ApplyFlag), []byte("1"), 0o644); err != nil {
		fail(w, 500, "could not confirm the restore")
		return
	}
	ok(w, map[string]any{"restarting": true})
	if s.Exit != nil {
		go func() { time.Sleep(700 * time.Millisecond); s.Exit() }()
	}
}
