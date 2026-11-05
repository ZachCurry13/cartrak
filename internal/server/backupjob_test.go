package server

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"
	"time"
)

func TestABackupIsMadeAsAJobAndTheFinishedFileDownloadsAtOnce(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	srv.DataDir = t.TempDir()
	admin := adminClient(t, ts.URL)
	xh := map[string]string{"X-Requested-With": "cartrak"}
	if code, _ := do(t, &http.Client{}, "POST", ts.URL+"/api/backup/start", "", xh); code != 401 {
		t.Fatalf("anonymous: %d", code)
	}
	code, body := do(t, admin, "POST", ts.URL+"/api/backup/start?small=1", "", xh)
	var job struct{ ID string }
	if code != 200 || json.Unmarshal([]byte(body), &job) != nil || job.ID == "" {
		t.Fatalf("start: %d %s", code, body)
	}
	var st struct{ State string }
	for i := 0; i < 100 && st.State != "ready"; i++ {
		_, b := do(t, admin, "GET", ts.URL+"/api/backup/status?id="+job.ID, "", nil)
		_ = json.Unmarshal([]byte(b), &st)
		time.Sleep(50 * time.Millisecond)
	}
	if st.State != "ready" {
		t.Fatalf("the copy never finished: %q", st.State)
	}
	file := download(t, admin, ts.URL+"/api/backup/file?id="+job.ID)
	if len(file) < 4096 || string(file[:15]) != "SQLite format 3" {
		t.Fatalf("a real database: %d bytes", len(file))
	}
	if code, _ := do(t, admin, "GET", ts.URL+"/api/backup/file?id="+job.ID, "", nil); code != 404 {
		t.Fatalf("a finished file is handed over once: %d", code)
	}
	time.Sleep(300 * time.Millisecond) // the handler tidies up just after the last byte is sent
	if left, _ := filepath.Glob(filepath.Join(srv.DataDir, "backup-*")); len(left) != 0 {
		t.Fatalf("the temporary copy is removed: %v", left)
	}
}
