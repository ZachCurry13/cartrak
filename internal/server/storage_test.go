package server

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestTrimNeedsTheWordAndAtLeastThirtyDaysAndRunsInTheBackground(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	old := time.Now().AddDate(0, 0, -200).Unix()
	for i := 0; i < 3; i++ {
		srv.S.DB.Exec(`INSERT INTO snapshots(ts,provider,raw) VALUES(?,?,?)`, old+int64(i), "t", `{"x":1}`)
		srv.S.DB.Exec(`INSERT INTO samples(ts,extra) VALUES(?,?)`, old+int64(i), `{"x":1}`)
	}
	post := func(body string) int { c, _ := do(t, admin, "POST", ts.URL+"/api/storage/trim", body, xh); return c }
	if post(`{"days":90,"confirm":"yes"}`) != 400 || post(`{"days":7,"confirm":"TRIM"}`) != 400 {
		t.Fatal("the word TRIM and at least 30 days are required")
	}
	_, b := do(t, admin, "GET", ts.URL+"/api/storage?days=90", "", nil)
	var info struct{ Snapshots, Samples int }
	json.Unmarshal([]byte(b), &info)
	if info.Snapshots != 3 || info.Samples != 3 {
		t.Fatalf("%s", b)
	}
	if post(`{"days":90,"confirm":"TRIM"}`) != 200 {
		t.Fatal("start")
	}
	deadline := time.Now().Add(10 * time.Second)
	for {
		_, b := do(t, admin, "GET", ts.URL+"/api/storage?days=90", "", nil)
		var st struct{ Trim struct{ Running bool } }
		json.Unmarshal([]byte(b), &st)
		if !st.Trim.Running {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the trim did not finish")
		}
		time.Sleep(100 * time.Millisecond)
	}
	var left int
	srv.S.DB.QueryRow(`SELECT COUNT(*) FROM snapshots`).Scan(&left)
	if left != 0 {
		t.Fatalf("raw copies removed: %d left", left)
	}
	if c, _ := do(t, &http.Client{}, "GET", ts.URL+"/api/storage", "", nil); c != 401 {
		t.Fatalf("anonymous: %d", c)
	}
}
