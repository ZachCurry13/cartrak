package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

func TestBugReportsAreKeptAsTextAndOpenedThroughOneRevocableLink(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	srv.Cfg.Set(keyAddress, "cartrak.example.test")
	anon := &http.Client{}
	admin := adminClient(t, ts.URL)
	api := ts.URL + "/api/bug-reports"
	if c, _ := do(t, admin, "POST", api, `{"note":"ring cut off","page":"#/trips","device":"phone 411x890","text":"Trips","layout":"scrolls sideways","debug":"dbg-line"}`, xh); c != 200 {
		t.Fatalf("add: %d", c)
	}
	_, out := do(t, admin, "GET", api, "", nil)
	var l struct {
		Reports []struct{ ID int64 }
		Link    *struct{}
	}
	if json.Unmarshal([]byte(out), &l) != nil || len(l.Reports) != 1 || l.Link != nil {
		t.Fatalf("list: %s", out)
	}
	_, out = do(t, admin, "POST", api+"/link", `{"days":7}`, xh)
	var link struct{ Path, URL string }
	if json.Unmarshal([]byte(out), &link) != nil || !strings.HasPrefix(link.Path, "/r/") || !strings.HasPrefix(link.URL, "https://cartrak.example.test/r/") {
		t.Fatalf("link: %s", out)
	}
	if c, body := do(t, anon, "GET", ts.URL+link.Path, "", nil); c != 200 || !strings.Contains(body, "ring cut off") || !strings.Contains(body, "scrolls sideways") || !strings.Contains(body, "dbg-line") {
		t.Fatalf("the link shows the reports as text: %d %s", c, body)
	}
	if c, body := do(t, anon, "GET", ts.URL+link.Path+"/"+strconv.FormatInt(l.Reports[0].ID, 10), "", nil); c != 200 || !strings.Contains(body, "ring cut off") {
		t.Fatalf("one report: %d %s", c, body)
	}
	if c, _ := do(t, anon, "GET", ts.URL+"/r/not-the-secret", "", nil); c != 404 {
		t.Fatalf("a wrong secret gave %d", c)
	}
	do(t, admin, "DELETE", api+"/link", "", xh)
	if c, _ := do(t, anon, "GET", ts.URL+link.Path, "", nil); c != 404 {
		t.Fatalf("a revoked link gave %d", c)
	}
	do(t, admin, "POST", api+"/clear", `{"olderThanDays":0}`, xh)
	if _, out = do(t, admin, "GET", api, "", nil); !strings.Contains(out, `"reports":[]`) {
		t.Fatalf("after clearing: %s", out)
	}
}
