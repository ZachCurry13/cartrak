package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestTheConnectorEndpointsAreAdminOnlyAndRefuseWhereTheyCannotWork(t *testing.T) {
	ts, _ := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	if code, _ := do(t, &http.Client{}, "GET", ts.URL+"/api/connector", "", nil); code != 401 {
		t.Fatalf("anonymous: %d", code)
	}
	code, body := do(t, admin, "GET", ts.URL+"/api/connector", "", nil)
	if code != 200 || !strings.Contains(body, `"supported"`) {
		t.Fatalf("info: %d %s", code, body)
	}
	if code, _ := do(t, admin, "POST", ts.URL+"/api/connector/update", "", xh); code != 400 {
		t.Fatalf("with no manager the update is refused: %d", code)
	}
	if code, _ := do(t, admin, "GET", ts.URL+"/api/about", "", nil); code != 200 {
		t.Fatalf("about: %d", code)
	}
}
