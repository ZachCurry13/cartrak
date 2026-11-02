package server

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
)

func TestSoftwareVersionNotesAreKeptNewestFirstAndCanBeRemoved(t *testing.T) {
	ts, _ := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	if _, empty := do(t, admin, "GET", ts.URL+"/api/software", "", nil); !strings.Contains(empty, `"notes":[]`) {
		t.Fatalf("no notes is an empty list, not null: %s", empty)
	}
	post := func(body string) int { c, _ := do(t, admin, "POST", ts.URL+"/api/software", body, xh); return c }
	if post(`{"date":"2026-11-02","version":""}`) != 400 || post(`{"date":"soon","version":"1"}`) != 400 || post(`{"date":"2999-01-01","version":"1"}`) != 400 {
		t.Fatal("an empty version, a bad date and a future date are refused")
	}
	if post(`{"date":"2026-01-05","version":"first"}`) != 200 || post(`{"date":"2026-10-20","version":"second"}`) != 200 {
		t.Fatal("adding notes")
	}
	type list struct{ Notes []softwareNote }
	var l list
	_, body := do(t, admin, "GET", ts.URL+"/api/software", "", nil)
	if json.Unmarshal([]byte(body), &l) != nil || len(l.Notes) != 2 || l.Notes[0].Version != "second" || l.Notes[0].ID == l.Notes[1].ID {
		t.Fatalf("newest first with their own ids: %s", body)
	}
	if c, _ := do(t, admin, "DELETE", ts.URL+"/api/software/"+strconv.FormatInt(l.Notes[0].ID, 10), "", xh); c != 200 {
		t.Fatal("remove")
	}
	_, body = do(t, admin, "GET", ts.URL+"/api/software", "", nil)
	l = list{}
	json.Unmarshal([]byte(body), &l)
	if len(l.Notes) != 1 || l.Notes[0].Version != "first" {
		t.Fatalf("after removing: %s", body)
	}
}
