package server

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"
	"time"
)

func zipFilesOf(t *testing.T, body string) map[string]string {
	zr, err := zip.NewReader(bytes.NewReader([]byte(body)), int64(len(body)))
	if err != nil {
		t.Fatalf("not a zip: %v", err)
	}
	out := map[string]string{}
	for _, f := range zr.File {
		rc, _ := f.Open()
		b, _ := io.ReadAll(rc)
		rc.Close()
		out[f.Name] = string(b)
	}
	return out
}

func TestTrackingRecordsOnlyWhileOnAndExportsWhatHappened(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)

	if code, _ := do(t, admin, "GET", ts.URL+"/api/track/export", "", nil); code != 404 {
		t.Fatalf("no export before a session: %d", code)
	}
	do(t, admin, "GET", ts.URL+"/api/trips", "", nil) // before tracking: not recorded
	if code, body := do(t, admin, "POST", ts.URL+"/api/track/start", `{"hours":2}`, xh); code != 200 || !strings.Contains(body, `"active":true`) {
		t.Fatalf("start: %d %s", code, body)
	}
	do(t, admin, "GET", ts.URL+"/api/trips?limit=3", "", nil)
	do(t, admin, "GET", ts.URL+"/api/track", "", nil) // the tracking calls themselves are not recorded
	evs := `{"events":[{"kind":"page","a":"#/trips","b":"#/"},{"kind":"tap","a":"Export CSV","b":"#/trips"},{"kind":"error","a":"boom","b":"#/day"},{"kind":"bogus","a":"x"}]}`
	if _, body := do(t, admin, "POST", ts.URL+"/api/track/events", evs, xh); !strings.Contains(body, `"stored":3`) {
		t.Fatalf("three good events kept, the bogus kind dropped: %s", body)
	}
	srv.S.LogDecision("setting", "changed: Gas price")
	do(t, admin, "POST", ts.URL+"/api/track/stop", "", xh)
	do(t, admin, "GET", ts.URL+"/api/trips?after=1", "", nil) // after the stop: not recorded
	if _, body := do(t, admin, "POST", ts.URL+"/api/track/events", evs, xh); !strings.Contains(body, `"stored":0`) {
		t.Fatalf("nothing is kept once it has stopped: %s", body)
	}

	code, body := do(t, admin, "GET", ts.URL+"/api/track/export", "", nil)
	if code != 200 {
		t.Fatalf("export: %d", code)
	}
	f := zipFilesOf(t, body)
	for _, name := range []string{"README.txt", "debug-report.txt", "readings-raw.jsonl", "readings.csv", "trips.jsonl", "charges.jsonl", "decisions.csv", "alerts.csv", "requests.csv", "app-events.csv", "field-check.txt"} {
		if _, ok := f[name]; !ok {
			t.Fatalf("the export lacks %s: %v", name, mapKeys(f))
		}
	}
	if !strings.Contains(f["requests.csv"], "GET /api/trips?limit=3") || strings.Contains(f["requests.csv"], "after=1") || strings.Contains(f["requests.csv"], "/api/track") {
		t.Fatalf("requests in the window only, not the tracking calls:\n%s", f["requests.csv"])
	}
	if !strings.Contains(f["app-events.csv"], "Export CSV") || !strings.Contains(f["app-events.csv"], "boom") || strings.Contains(f["app-events.csv"], "bogus") {
		t.Fatalf("app events: %s", f["app-events.csv"])
	}
	if !strings.Contains(f["decisions.csv"], "changed: Gas price") || !strings.Contains(f["decisions.csv"], "tracking") {
		t.Fatalf("decisions: %s", f["decisions.csv"])
	}
	if !strings.Contains(f["README.txt"], "left out") {
		t.Fatalf("the export says it is without positions: %s", f["README.txt"])
	}
	_, body = do(t, admin, "GET", ts.URL+"/api/track/export?private=1", "", nil)
	if !strings.Contains(zipFilesOf(t, body)["README.txt"], "INCLUDED") {
		t.Fatal("the private export says it holds positions")
	}
	_ = time.Second
}

func mapKeys(m map[string]string) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}

func TestFieldCheckAnswersAreKeptAndShowInTheDebugReport(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	st := `{"state":{"at":1700000000,"checks":{"badge":{"s":"bad","note":"no count"}},"auto":[["Savings page speed","40 ms"]],"titles":{"badge":"App icon badge"}}}`
	if code, _ := do(t, admin, "PUT", ts.URL+"/api/fieldcheck", st, xh); code != 200 {
		t.Fatalf("save: %d", code)
	}
	if _, body := do(t, admin, "GET", ts.URL+"/api/fieldcheck", "", nil); !strings.Contains(body, "no count") {
		t.Fatalf("read back: %s", body)
	}
	text, _, err := srv.DebugText(24, false)
	if err != nil || !strings.Contains(text, "FIELD CHECK") || !strings.Contains(text, "App icon badge: PROBLEM | no count") || !strings.Contains(text, "Savings page speed: 40 ms") {
		t.Fatalf("the Debug report carries the answers (%v):\n%s", err, text[max(0, len(text)-600):])
	}
}
