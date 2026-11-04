package server

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
)

func TestSavedTripsKeepOpenAndDelete(t *testing.T) {
	ts, _ := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	if c, _ := do(t, admin, "POST", ts.URL+"/api/saved-trips", `{"name":"","body":{}}`, xh); c != 400 {
		t.Fatalf("an unnamed trip gave %d", c)
	}
	if c, _ := do(t, admin, "POST", ts.URL+"/api/saved-trips", `{"name":"Up north","body":{"dest":"x"}}`, xh); c != 200 {
		t.Fatalf("save gave %d", c)
	}
	_, out := do(t, admin, "GET", ts.URL+"/api/saved-trips", "", nil)
	var l []struct {
		ID   int64
		Name string
		Body string
	}
	if json.Unmarshal([]byte(out), &l) != nil || len(l) != 1 || l[0].Name != "Up north" || !strings.Contains(l[0].Body, `"dest"`) {
		t.Fatalf("list: %s", out)
	}
	do(t, admin, "DELETE", ts.URL+"/api/saved-trips/"+strconv.FormatInt(l[0].ID, 10), "", xh)
	if _, out = do(t, admin, "GET", ts.URL+"/api/saved-trips", "", nil); strings.TrimSpace(out) != "[]" {
		t.Fatalf("after delete: %s", out)
	}
}
