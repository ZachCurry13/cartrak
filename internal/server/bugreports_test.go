package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
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
	if c, body := do(t, anon, "GET", ts.URL+link.Path, "", nil); c != 200 || !strings.Contains(body, "ring cut off") || !strings.Contains(body, "scrolls sideways") || strings.Contains(body, "dbg-line") {
		t.Fatalf("the index is short: the note and the layout facts, not the debug report: %d %s", c, body)
	}
	id := strconv.FormatInt(l.Reports[0].ID, 10)
	if c, body := do(t, anon, "GET", ts.URL+link.Path+"/"+id, "", nil); c != 200 || !strings.Contains(body, "ring cut off") || strings.Contains(body, "dbg-line") || !strings.Contains(body, "/"+id+"/debug") {
		t.Fatalf("one report points to its debug report instead of holding it: %d %s", c, body)
	}
	if c, body := do(t, anon, "GET", ts.URL+link.Path+"/"+id+"/debug", "", nil); c != 200 || !strings.Contains(body, "dbg-line") {
		t.Fatalf("the debug report is at its own address: %d %s", c, body)
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

func TestAReportsPageCopyIsServedWithAPolicyThatRunsNothing(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	srv.Cfg.Set(keyAddress, "cartrak.example.test")
	admin := adminClient(t, ts.URL)
	anon := &http.Client{}
	api := ts.URL + "/api/bug-reports"
	do(t, admin, "POST", api, `{"note":"layout","page":"#/plan","device":"phone 411x800","html":"<!doctype html><html><body><p>copy</p></body></html>"}`, xh)
	do(t, admin, "POST", api, `{"note":"no copy","page":"#/plan","device":"phone"}`, xh)
	_, out := do(t, admin, "POST", api+"/link", `{"days":7}`, xh)
	var link struct{ Path string }
	json.Unmarshal([]byte(out), &link)
	_, body := do(t, admin, "GET", api, "", nil)
	var l struct {
		Reports []struct {
			ID      int64
			HasPage bool
		}
		LinkURL string
	}
	if json.Unmarshal([]byte(body), &l) != nil || !strings.HasSuffix(l.LinkURL, link.Path) || len(l.Reports) != 2 || !l.Reports[1].HasPage || l.Reports[0].HasPage {
		t.Fatalf("the list shows the link again and which reports have a page copy: %s", body)
	}
	req, _ := http.NewRequest("GET", ts.URL+link.Path+"/"+strconv.FormatInt(l.Reports[1].ID, 10)+"/page", nil)
	resp, err := anon.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 || !strings.Contains(resp.Header.Get("Content-Security-Policy"), "default-src 'none'") || resp.Header.Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatalf("the copy must carry a policy that runs nothing: %d %v", resp.StatusCode, resp.Header)
	}
	if c, _ := do(t, anon, "GET", ts.URL+link.Path+"/"+strconv.FormatInt(l.Reports[0].ID, 10)+"/page", "", nil); c != 404 {
		t.Fatalf("a report with no copy gave %d", c)
	}
	if _, text := do(t, anon, "GET", ts.URL+link.Path, "", nil); !strings.Contains(text, "/page") {
		t.Fatalf("the text index points to the page copy: %s", text)
	}
}

func TestAFixedReportLeavesTheIndexAndIsClearedAWeekLater(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	srv.Cfg.Set(keyAddress, "cartrak.example.test")
	admin := adminClient(t, ts.URL)
	anon := &http.Client{}
	api := ts.URL + "/api/bug-reports"
	do(t, admin, "POST", api, `{"note":"first problem","page":"#/a","device":"phone"}`, xh)
	do(t, admin, "POST", api, `{"note":"second problem","page":"#/b","device":"phone"}`, xh)
	_, out := do(t, admin, "POST", api+"/link", `{"days":7}`, xh)
	var link struct{ Path string }
	json.Unmarshal([]byte(out), &link)
	if c, _ := do(t, anon, "POST", ts.URL+link.Path+"/1/fixed", "", nil); c != 200 {
		t.Fatalf("marking a report fixed through the link gave %d", c)
	}
	if c, _ := do(t, anon, "POST", ts.URL+"/r/not-the-secret/1/fixed", "", nil); c != 404 {
		t.Fatalf("a wrong secret must not mark anything: %d", c)
	}
	if _, idx := do(t, anon, "GET", ts.URL+link.Path, "", nil); strings.Contains(idx, "first problem") || !strings.Contains(idx, "second problem") || !strings.Contains(idx, "1 fixed") {
		t.Fatalf("the index lists only open reports and counts the fixed: %s", idx)
	}
	if c, _ := do(t, admin, "PUT", api+"/1/fixed", `{"fixed":false}`, xh); c != 200 {
		t.Fatalf("reopening: %d", c)
	}
	if _, idx := do(t, anon, "GET", ts.URL+link.Path, "", nil); !strings.Contains(idx, "first problem") {
		t.Fatalf("an open report is back in the index: %s", idx)
	}
	do(t, admin, "PUT", api+"/1/fixed", `{"fixed":true}`, xh)
	if _, b := do(t, admin, "POST", api+"/clear-fixed", "", xh); !strings.Contains(b, `"removed":1`) {
		t.Fatalf("clear fixed: %s", b)
	}
	if _, b := do(t, admin, "GET", api, "", nil); strings.Contains(b, "first problem") || !strings.Contains(b, "second problem") {
		t.Fatalf("only the open one is left: %s", b)
	}
	// a report fixed more than a week ago goes by itself
	if err := srv.S.SetBugReportFixed(2, time.Now().Add(-8*24*time.Hour).Unix()); err != nil {
		t.Fatal(err)
	}
	if _, b := do(t, admin, "GET", api, "", nil); strings.Contains(b, "second problem") {
		t.Fatalf("a report fixed over a week ago is cleared: %s", b)
	}
}
