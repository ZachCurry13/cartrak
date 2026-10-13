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
