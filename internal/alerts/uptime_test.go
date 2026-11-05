package alerts

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCarTrakSaysWhenItIsBackAfterBeingOfflineForMoreThanAFewMinutes(t *testing.T) {
	e, r, st := setup(t)
	now := time.Now()
	e.Now = func() time.Time { return now }

	e.upCheck() // the very first run: nothing to compare with
	if len(r.got) != 0 {
		t.Fatalf("a first run is not a return: %v", r.kinds())
	}
	if st.GetKV("alive_ts") == "" {
		t.Fatal("it must note that it is running")
	}
	now = now.Add(2 * time.Minute) // a restart to update takes this long
	e.upCheck()
	if len(r.got) != 0 {
		t.Fatalf("two minutes is a restart, not an outage: %v", r.kinds())
	}
	_ = st.SetKV("alive_ts", fmt.Sprint(now.Add(-40*time.Minute).Unix()))
	e.upCheck()
	if r.kinds() != "server_back" || !strings.Contains(r.got[0].Body, "40 minutes") {
		t.Fatalf("a long gap is an outage, and the message says how long: %v %v", r.kinds(), r.got)
	}
	e.upCheck()
	if len(r.got) != 1 {
		t.Fatalf("it says so once: %v", r.kinds())
	}
}

func TestTheUptimeMonitorIsAskedEveryFiveMinutesAndTheResultIsKept(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits++ }))
	defer srv.Close()
	e, _, st := setup(t)
	now := time.Now()
	e.Now = func() time.Time { return now }
	e.HeartbeatURL = func() string { return srv.URL }
	e.beat()
	e.beat()
	if hits != 1 {
		t.Fatalf("asked once, not on every minute: %d", hits)
	}
	now = now.Add(6 * time.Minute)
	e.beat()
	if hits != 2 {
		t.Fatalf("asked again after five minutes: %d", hits)
	}
	if !strings.Contains(st.GetKV("heartbeat_last"), `"ok":true`) {
		t.Fatalf("how it went is kept: %s", st.GetKV("heartbeat_last"))
	}
	e.HeartbeatURL = func() string { return "" }
	now = now.Add(10 * time.Minute)
	e.beat()
	if hits != 2 {
		t.Fatalf("with no address nothing is asked: %d", hits)
	}
}
