package server

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ZachCurry13/cartrak/internal/settings"
)

func TestABugReportNeedsTheRepositoryAdminsAndASensibleSize(t *testing.T) {
	var gotAuth, issueTitle string
	var puts int
	gh := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		b, _ := io.ReadAll(r.Body)
		if r.Method == http.MethodPut {
			puts++
			_, _ = w.Write([]byte(`{"content":{"html_url":"https://example.test/x"}}`))
			return
		}
		var in map[string]any
		_ = json.Unmarshal(b, &in)
		issueTitle, _ = in["title"].(string)
		_, _ = w.Write([]byte(`{"number":3,"html_url":"https://example.test/issues/3"}`))
	}))
	defer gh.Close()
	ts, srv := healthServer(t, fakeProv{}, false, false)
	srv.ReportAPI = gh.URL
	admin := adminClient(t, ts.URL)
	png := "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("\x89PNG\r\n\x1a\nxxxx"))
	body := func(shot, note string) string {
		b, _ := json.Marshal(map[string]string{"note": note, "page": "#/trips", "device": "phone 411x890", "text": "Trips", "debug": "dbg", "screenshot": shot})
		return string(b)
	}
	url := ts.URL + "/api/bug-report"

	if code, b := do(t, admin, "GET", url+"/status", "", nil); code != 200 || !strings.Contains(b, `"configured":false`) {
		t.Fatalf("not set up yet: %d %s", code, b)
	}
	if code, _ := do(t, admin, "POST", url, body(png, "ring cut off"), xh); code != 409 {
		t.Fatalf("without a repository: %d", code)
	}
	srv.Cfg.Set(settings.ReportRepoKey, "me/reports")
	srv.Cfg.Set(settings.ReportTokenKey, "ghp_x")
	if code, b := do(t, admin, "GET", url+"/status", "", nil); code != 200 || !strings.Contains(b, `"configured":true`) || strings.Contains(b, "ghp_x") {
		t.Fatalf("set up, and the token is not shown: %d %s", code, b)
	}
	code, b := do(t, admin, "POST", url, body(png, "ring cut off"), xh)
	if code != 200 || !strings.Contains(b, `"number":3`) || puts != 3 || gotAuth != "Bearer ghp_x" || issueTitle == "" {
		t.Fatalf("sent: %d %s puts=%d auth=%q title=%q", code, b, puts, gotAuth, issueTitle)
	}
	if code, _ := do(t, admin, "POST", url, body("data:image/jpeg;base64,AAAA", "x"), xh); code != 400 {
		t.Fatalf("only a PNG: %d", code)
	}
	if code, _ := do(t, admin, "POST", url, body("data:image/png;base64,not base64!!", "x"), xh); code != 400 {
		t.Fatalf("a broken picture: %d", code)
	}
	if code, _ := do(t, admin, "POST", url, body(png, strings.Repeat("x", 5000)), xh); code != 400 {
		t.Fatalf("a note that is far too long: %d", code)
	}
	if code, _ := do(t, admin, "POST", url, body("", "no picture is fine"), xh); code != 200 {
		t.Fatalf("a report without a screenshot: %d", code)
	}
	do(t, admin, "POST", ts.URL+"/api/users", `{"name":"fam","password":"viewer-password-1","role":"viewer"}`, xh)
	vj, _ := cookieJar()
	v := &http.Client{Jar: vj}
	do(t, v, "POST", ts.URL+"/api/login", `{"name":"fam","password":"viewer-password-1"}`, nil)
	if code, _ := do(t, v, "POST", url, body(png, "x"), xh); code != 403 {
		t.Fatalf("a viewer cannot send the car's data off the box: %d", code)
	}
}
