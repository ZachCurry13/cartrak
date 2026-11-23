package server

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
)

func TestASavedRouteIsUpdatedInPlaceNotCopied(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	code, body := do(t, admin, "POST", ts.URL+"/api/saved-trips", `{"name":"Monday","body":{"kind":"route","stops":[{"name":"A"},{"name":"B"}]}}`, xh)
	var made struct{ ID int64 }
	if code != 200 || json.Unmarshal([]byte(body), &made) != nil || made.ID == 0 {
		t.Fatalf("%d %s", code, body)
	}
	url := ts.URL + "/api/saved-trips/" + strconv.FormatInt(made.ID, 10)
	if c, b := do(t, admin, "PUT", url, `{"name":"Monday","body":{"kind":"route","stops":[{"name":"A"},{"name":"B"},{"name":"Costco"}]}}`, xh); c != 200 {
		t.Fatalf("update: %d %s", c, b)
	}
	all, _ := srv.S.SavedTrips()
	if len(all) != 1 || !strings.Contains(all[0].Body, "Costco") {
		t.Fatalf("one route, with the new stop: %+v", all)
	}
	if c, _ := do(t, admin, "PUT", ts.URL+"/api/saved-trips/999", `{"name":"x","body":{"kind":"route"}}`, xh); c != 404 {
		t.Fatalf("a route that is gone: %d", c)
	}
	if c, _ := do(t, admin, "PUT", url, `{"name":"","body":{}}`, xh); c != 400 {
		t.Fatalf("a name is needed: %d", c)
	}
}
