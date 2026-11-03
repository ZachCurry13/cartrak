package bugreport

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeGitHub records what is sent and answers like the real thing.
type fakeGitHub struct {
	mu    sync.Mutex
	puts  map[string][]byte
	issue map[string]any
	auth  []string
	fail  int
}

func (f *fakeGitHub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.auth = append(f.auth, r.Header.Get("Authorization"))
	if f.fail != 0 {
		w.WriteHeader(f.fail)
		_, _ = w.Write([]byte(`{"message":"Bad credentials"}`))
		return
	}
	b, _ := io.ReadAll(r.Body)
	switch {
	case r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/contents/reports/"):
		var in struct{ Content string }
		_ = json.Unmarshal(b, &in)
		data, _ := base64.StdEncoding.DecodeString(in.Content)
		f.puts[r.URL.Path] = data
		_, _ = w.Write([]byte(`{"content":{"html_url":"https://example.test/blob/main` + r.URL.Path + `"}}`))
	case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/issues"):
		_ = json.Unmarshal(b, &f.issue)
		_, _ = w.Write([]byte(`{"number":7,"html_url":"https://example.test/issues/7"}`))
	default:
		w.WriteHeader(404)
	}
}

func TestAReportBecomesFilesAndAnIssueAndTheTokenOnlyGoesInTheHeader(t *testing.T) {
	f := &fakeGitHub{puts: map[string][]byte{}}
	srv := httptest.NewServer(f)
	defer srv.Close()
	c := &Client{Repo: "me/reports", Token: "ghp_secrettoken", API: srv.URL}
	png := []byte("\x89PNG fake")
	res, err := c.Send(context.Background(), Report{Version: "9.9.9", Note: "The ring is cut off\nsecond line", Page: "#/trips", Device: "phone 411x890, touch, installed app",
		PageText: "Trips ``` inside", Debug: "debug text", Screenshot: png}, time.Date(2026, 11, 3, 15, 30, 45, 0, time.UTC))
	if err != nil || res.Number != 7 || res.URL != "https://example.test/issues/7" || !strings.HasPrefix(res.ID, "20261103-153045-") {
		t.Fatalf("%+v %v", res, err)
	}
	var names []string
	for p := range f.puts {
		names = append(names, p[strings.LastIndex(p, "/")+1:])
	}
	if len(names) != 3 {
		t.Fatalf("report.md, debug.txt and screen.png: %v", names)
	}
	for p, data := range f.puts {
		if strings.HasSuffix(p, "screen.png") && string(data) != string(png) {
			t.Error("the screenshot is stored as it was sent")
		}
		if strings.HasSuffix(p, "report.md") {
			s := string(data)
			if !strings.Contains(s, "The ring is cut off") || !strings.Contains(s, "phone 411x890") || strings.Contains(s, "``` inside") {
				t.Errorf("the note, the screen and the page text (fenced safely): %s", s)
			}
		}
	}
	if title, _ := f.issue["title"].(string); title != "[v9.9.9] The ring is cut off" {
		t.Errorf("the title is the first line of the note: %q", title)
	}
	for _, a := range f.auth {
		if a != "Bearer ghp_secrettoken" {
			t.Errorf("every call is signed: %q", a)
		}
	}
	all, _ := json.Marshal(f.issue)
	for _, d := range f.puts {
		all = append(all, d...)
	}
	if strings.Contains(string(all), "ghp_secrettoken") {
		t.Fatal("the token is never in what is stored")
	}
}

func TestAReportWithoutAScreenshotOrDebugTextSendsJustTheNote(t *testing.T) {
	f := &fakeGitHub{puts: map[string][]byte{}}
	srv := httptest.NewServer(f)
	defer srv.Close()
	if _, err := (&Client{Repo: "me/reports", Token: "t", API: srv.URL}).Send(context.Background(), Report{Version: "1", Note: "n"}, time.Now()); err != nil || len(f.puts) != 1 {
		t.Fatalf("%v %d", err, len(f.puts))
	}
}

func TestGitHubErrorsAreExplainedAndNeverShowTheToken(t *testing.T) {
	for code, want := range map[int]string{401: "did not accept the token", 404: "can write Contents and Issues in me/reports", 422: "GitHub said 422"} {
		f := &fakeGitHub{puts: map[string][]byte{}, fail: code}
		srv := httptest.NewServer(f)
		_, err := (&Client{Repo: "me/reports", Token: "ghp_secrettoken", API: srv.URL}).Send(context.Background(), Report{Version: "1", Note: "n"}, time.Now())
		srv.Close()
		if err == nil || !strings.Contains(err.Error(), want) || strings.Contains(err.Error(), "ghp_secrettoken") {
			t.Errorf("%d: %v", code, err)
		}
	}
}
