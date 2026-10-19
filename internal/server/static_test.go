package server

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestTheAppIsServedFromVersionedAddressesSoNoCacheCanKeepAnOldCopy(t *testing.T) {
	ts, _ := healthServer(t, fakeProv{}, false, false) // version 1.0.0
	get := func(path string) (*http.Response, string) {
		resp, err := http.Get(ts.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return resp, string(b)
	}
	resp, page := get("/")
	if !strings.Contains(page, `src="/v1.0.0/js/app.js"`) || !strings.Contains(page, `href="/v1.0.0/css/look.css"`) || strings.Contains(page, `src="/js/`) {
		t.Fatalf("the page points at versioned files: %s", page)
	}
	if resp.Header.Get("Cache-Control") != "no-cache" {
		t.Fatalf("the page itself is never cached: %q", resp.Header.Get("Cache-Control"))
	}
	resp, js := get("/v1.0.0/js/app.js")
	if resp.StatusCode != 200 || !strings.Contains(js, "import") {
		t.Fatalf("%d", resp.StatusCode)
	}
	if cc := resp.Header.Get("Cache-Control"); !strings.Contains(cc, "immutable") {
		t.Fatalf("a versioned file may be kept for good: %q", cc)
	}
	if resp, inner := get("/v1.0.0/js/pages/live.js"); resp.StatusCode != 200 || !strings.Contains(inner, "export default") {
		t.Fatalf("modules imported by relative paths resolve under the same version: %d", resp.StatusCode)
	}
	// A stale copy of the page that still points at an older version gets the current file, uncached.
	resp, old := get("/v0.0.1/js/app.js")
	if resp.StatusCode != 200 || old != js || strings.Contains(resp.Header.Get("Cache-Control"), "immutable") {
		t.Fatalf("an old address still works: %d %q", resp.StatusCode, resp.Header.Get("Cache-Control"))
	}
	resp, sw := get("/sw.js")
	if strings.Contains(sw, "__V__") || !strings.Contains(sw, "const V = '1.0.0'") || resp.Header.Get("Cache-Control") != "no-cache" || !strings.Contains(resp.Header.Get("Content-Type"), "javascript") {
		t.Fatalf("the service worker carries the version and is never cached: %q %s", resp.Header.Get("Cache-Control"), sw[:120])
	}
	if resp, _ := get("/icons/icon.svg"); resp.StatusCode != 200 {
		t.Fatalf("other files still served: %d", resp.StatusCode)
	}
	if resp, _ := get("/api/health"); resp.StatusCode == 200 {
		t.Fatal("the API is not static")
	}
}
