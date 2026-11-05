package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestADownloadedListIsCheckedForANewerOneAndUpdatedOnRequest(t *testing.T) {
	version, sites := `"v1"`, `[{"name":"First","status":"OPEN","gps":{"latitude":45,"longitude":-93},"powerKilowatt":250,"otherEVs":true,"plugs":{"nacs":4}}]`
	svc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", version)
		if r.Method != http.MethodHead {
			_, _ = w.Write([]byte(sites))
		}
	}))
	defer svc.Close()
	ts, srv := healthServer(t, fakeProv{}, false, false)
	srv.SuperURL = svc.URL
	admin := adminClient(t, ts.URL)
	type view struct {
		Items []struct {
			ID        string
			Count     int
			Available bool
			Note      string
			UpdatedAt int64
		}
	}
	get := func(method, path string) view {
		c, b := do(t, admin, method, ts.URL+path, "", xh)
		var v view
		if c != 200 || json.Unmarshal([]byte(b), &v) != nil {
			t.Fatalf("%s %s: %d %s", method, path, c, b)
		}
		return v
	}
	super := func(v view) (n int, avail bool) {
		for _, it := range v.Items {
			if it.ID == "super" {
				return it.Count, it.Available
			}
		}
		t.Fatal("no Supercharger list in the view")
		return
	}
	if n, a := super(get("GET", "/api/data-updates")); n != 0 || a {
		t.Fatalf("nothing downloaded yet, so nothing to update: %d %v", n, a)
	}
	if c, _ := do(t, admin, "POST", ts.URL+"/api/data-updates/super/update", "", xh); c != 200 {
		t.Fatalf("the first download: %d", c)
	}
	if n, a := super(get("POST", "/api/data-updates/check")); n != 1 || a {
		t.Fatalf("just downloaded: current: %d %v", n, a)
	}
	attention := func() int {
		_, b := do(t, admin, "GET", ts.URL+"/api/integrations/attention", "", nil)
		var a struct{ Attention int }
		json.Unmarshal([]byte(b), &a)
		return a.Attention
	}
	before := attention()
	version, sites = `"v2"`, `[{"name":"First","status":"OPEN","gps":{"latitude":45,"longitude":-93},"powerKilowatt":250,"otherEVs":true,"plugs":{"nacs":4}},{"name":"Second","status":"OPEN","gps":{"latitude":46,"longitude":-93},"powerKilowatt":250,"otherEVs":true,"plugs":{"nacs":4}}]`
	if _, a := super(get("POST", "/api/data-updates/check")); !a {
		t.Fatal("the list changed upstream: an update should be offered")
	}
	if attention() != before+1 {
		t.Fatalf("an update available adds one to the dot on More: %d then %d", before, attention())
	}
	if n, a := super(get("POST", "/api/data-updates/super/update")); n != 2 || a {
		t.Fatalf("after updating there are two sites and nothing newer: %d %v", n, a)
	}
	if attention() != before {
		t.Fatalf("after updating the dot is back to what it was")
	}
	if c, _ := do(t, admin, "POST", ts.URL+"/api/data-updates/nope/update", "", xh); c != 404 {
		t.Fatalf("an unknown list gave %d", c)
	}
	if c, _ := do(t, admin, "POST", ts.URL+"/api/data-updates/ea/update", "", xh); c == 200 {
		t.Fatal("the Electrify America list cannot be updated without its key")
	}
}
