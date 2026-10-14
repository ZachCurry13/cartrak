package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
)

func TestTheSignInLogAndSignedInDevicesCanBeSeenAndRevoked(t *testing.T) {
	ts, _ := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	bad := &http.Client{}
	if code, _ := do(t, bad, "POST", ts.URL+"/api/login", `{"name":"owner","password":"not-the-password"}`, map[string]string{"User-Agent": "Mozilla/5.0 (Linux; Android 14) Chrome/120.0 Mobile Safari/537.36"}); code != 401 {
		t.Fatalf("wrong password: %d", code)
	}
	code, b := do(t, admin, "GET", ts.URL+"/api/security", "", nil)
	var out struct {
		Signins []struct {
			Name  string
			OK    bool
			Agent string
		}
		Sessions []struct {
			ID      int64
			User    string
			Current bool
		}
	}
	if code != 200 || json.Unmarshal([]byte(b), &out) != nil {
		t.Fatalf("%d %s", code, b)
	}
	if len(out.Signins) == 0 || out.Signins[0].Name != "owner" || out.Signins[0].OK || out.Signins[0].Agent != "Chrome on Android" {
		t.Fatalf("the failed attempt is logged, readable and without any password: %s", b)
	}
	if strings.Contains(b, "not-the-password") {
		t.Fatal("the password must never be stored")
	}
	if len(out.Sessions) == 0 || !out.Sessions[0].Current {
		t.Fatalf("this device is marked as the current one: %s", b)
	}
	if code, _ := do(t, &http.Client{}, "GET", ts.URL+"/api/security", "", nil); code != 401 {
		t.Fatalf("anonymous: %d", code)
	}
	if code, _ := do(t, admin, "DELETE", fmt.Sprintf("%s/api/security/sessions/%d", ts.URL, out.Sessions[0].ID), "", xh); code != 200 {
		t.Fatalf("revoke: %d", code)
	}
	if code, _ := do(t, admin, "GET", ts.URL+"/api/security", "", nil); code != 401 {
		t.Fatalf("the revoked device is signed out: %d", code)
	}
}

func TestABackupIsAConsistentCopyWithoutSessionsAndTheSmallOneDropsRawReadings(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	srv.DataDir = t.TempDir()
	admin := adminClient(t, ts.URL)
	raw, _ := os.ReadFile("../ford/testdata/bev_response.json")
	if _, err := srv.S.SaveState("t", ford.ParseState(raw, time.Now())); err != nil {
		t.Fatal(err)
	}
	full, small := download(t, admin, ts.URL+"/api/backup"), download(t, admin, ts.URL+"/api/backup?small=1")
	if len(full) < 4096 || string(full[:15]) != "SQLite format 3" || len(small) >= len(full) {
		t.Fatalf("a real database, and the small one is smaller: %d vs %d", len(full), len(small))
	}
	left, _ := filepath.Glob(filepath.Join(srv.DataDir, "backup-*"))
	if len(left) != 0 {
		t.Fatalf("temporary copies are removed: %v", left)
	}
	if code, _ := do(t, &http.Client{}, "GET", ts.URL+"/api/backup", "", nil); code != 401 {
		t.Fatalf("anonymous: %d", code)
	}
}

func download(t *testing.T, c *http.Client, url string) []byte {
	resp, err := c.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		t.Fatalf("%s: %d %s", url, resp.StatusCode, b)
	}
	return b
}

func TestARestoreIsUploadedCheckedConfirmedAndOnlyThenStopsTheApp(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	srv.DataDir = t.TempDir()
	stopped := make(chan bool, 1)
	srv.Exit = func() { stopped <- true }
	admin := adminClient(t, ts.URL)

	// A real backup of this server's own database, so it has the account the test signs in with.
	backup := download(t, admin, ts.URL+"/api/backup")
	post := func(path string, body []byte) (int, string) {
		req, _ := http.NewRequest("POST", ts.URL+path, strings.NewReader(string(body)))
		req.Header.Set("X-Requested-With", "cartrak")
		resp, err := admin.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, string(b)
	}
	if code, b := post("/api/restore/upload", []byte("not a database")); code != 400 || !strings.Contains(b, "not a CarTrak backup") {
		t.Fatalf("junk: %d %s", code, b)
	}
	if code, b := post("/api/restore/upload", backup); code != 200 || !strings.Contains(b, `"users"`) {
		t.Fatalf("upload: %d %s", code, b)
	}
	if code, _ := do(t, admin, "POST", ts.URL+"/api/restore/apply", `{"confirm":"yes"}`, xh); code != 400 {
		t.Fatalf("the word RESTORE is required: %d", code)
	}
	select {
	case <-stopped:
		t.Fatal("it must not stop without confirmation")
	default:
	}
	if code, _ := do(t, admin, "POST", ts.URL+"/api/restore/apply", `{"confirm":"RESTORE"}`, xh); code != 200 {
		t.Fatalf("apply: %d", code)
	}
	select {
	case <-stopped:
	case <-time.After(3 * time.Second):
		t.Fatal("after confirming, the app stops so the restore is applied at the next start")
	}
	if _, err := os.Stat(filepath.Join(srv.DataDir, "restore", "APPLY")); err != nil {
		t.Fatal("the confirmation is recorded for the next start")
	}
	if code, _ := do(t, &http.Client{}, "POST", ts.URL+"/api/restore/apply", `{"confirm":"RESTORE"}`, xh); code != 401 {
		t.Fatalf("anonymous: %d", code)
	}
}
