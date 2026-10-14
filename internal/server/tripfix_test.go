package server

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

func TestAnAdminCanDeleteAFinishedTripAndCloseAStuckOne(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	mk := func(startAgo, endAgo time.Duration, open bool) int64 {
		end := time.Now().Add(-endAgo).Unix()
		tr := &store.Trip{StartTS: time.Now().Add(-startAgo).Unix(), Category: "personal", CategorySource: "default"}
		if err := srv.S.CreateTrip(tr); err != nil {
			t.Fatal(err)
		}
		tr.EndTS, tr.Open = &end, open
		if err := srv.S.UpdateTrip(tr); err != nil {
			t.Fatal(err)
		}
		return tr.ID
	}
	done, stuck, live := mk(5*time.Hour, 4*time.Hour, false), mk(10*time.Hour, 9*time.Hour, true), mk(20*time.Minute, time.Minute, true)
	url := func(id int64, suffix string) string { return fmt.Sprintf("%s/api/trips/%d%s", ts.URL, id, suffix) }

	if code, _ := do(t, admin, "DELETE", url(stuck, ""), "", xh); code != 409 {
		t.Fatalf("an open trip cannot be deleted: %d", code)
	}
	if code, _ := do(t, admin, "POST", url(live, "/close"), "", xh); code != 409 {
		t.Fatalf("a trip with fresh readings is in progress: %d", code)
	}
	if code, _ := do(t, admin, "POST", url(done, "/close"), "", xh); code != 409 {
		t.Fatalf("already finished: %d", code)
	}
	if code, b := do(t, admin, "POST", url(stuck, "/close"), "", xh); code != 200 {
		t.Fatalf("close: %d %s", code, b)
	}
	if got, _ := srv.S.GetTrip(stuck); got == nil || got.Open {
		t.Fatal("it is closed now")
	}
	if code, b := do(t, admin, "DELETE", url(stuck, ""), "", xh); code != 200 {
		t.Fatalf("delete: %d %s", code, b)
	}
	if _, err := srv.S.GetTrip(stuck); err == nil {
		t.Fatal("it is gone")
	}
	if code, _ := do(t, admin, "DELETE", url(9999, ""), "", xh); code != 404 {
		t.Fatalf("no such trip: %d", code)
	}
	// A viewer can do neither.
	do(t, admin, "POST", ts.URL+"/api/users", `{"name":"fam","password":"viewer-password-1","role":"viewer"}`, xh)
	vj, _ := cookieJar()
	v := &http.Client{Jar: vj}
	do(t, v, "POST", ts.URL+"/api/login", `{"name":"fam","password":"viewer-password-1"}`, nil)
	if code, _ := do(t, v, "DELETE", url(done, ""), "", xh); code != 403 {
		t.Fatalf("viewer delete: %d", code)
	}
}
