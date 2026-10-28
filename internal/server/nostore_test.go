package server

import (
	"net/http"
	"testing"
)

func TestApiAnswersAreNotCachedButMapTilesStillAre(t *testing.T) {
	ts, _ := healthServer(t, fakeProv{}, false, false)
	for _, path := range []string{"/api/setup", "/api/me"} {
		resp, err := http.Get(ts.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if got := resp.Header.Get("Cache-Control"); got != "no-store" {
			t.Fatalf("%s: Cache-Control %q, want no-store", path, got)
		}
	}
}
