package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
)

func TestTheDebugReportListsReadingsAndLockFieldsButNeverPositionsOrTheVIN(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	lat, lon, soc := 44.123456, -93.654321, 70.0
	locked, unlocked := true, false
	mk := func(min int, l *bool, ign string) {
		st := &ford.State{TS: time.Now().Add(time.Duration(-min) * time.Minute), SoCPct: &soc, Lat: &lat, Lon: &lon, Locked: l, Ignition: ign,
			Extra: map[string]any{"doorLockStatus[0].value": "UNLOCKED", "doorLockStatus[0].vehicledoor": "UNSPECIFIED_FRONT", "vin": "SECRETVIN123", "position.latitude": 44.123456}, Raw: []byte(`{}`)}
		if _, err := srv.S.SaveState("t", st); err != nil {
			t.Fatal(err)
		}
	}
	mk(30, &locked, "OFF")
	mk(20, &locked, "OFF")
	mk(10, &unlocked, "OFF")
	code, b := do(t, admin, "GET", ts.URL+"/api/debug-report?hours=2", "", nil)
	var out struct {
		Text     string
		Readings int
	}
	if code != 200 || json.Unmarshal([]byte(b), &out) != nil || out.Readings != 3 {
		t.Fatalf("%d %s", code, b)
	}
	for _, want := range []string{"CarTrak debug report", "READINGS", "locked", "UNLOCKED", "LOCK AND DOOR FIELDS", "doorLockStatus[0].value = UNLOCKED", "TRIPS", "CHARGING", "ALERTS SENT", "the same 1 more time"} {
		if !strings.Contains(out.Text, want) {
			t.Errorf("missing %q in:\n%s", want, out.Text)
		}
	}
	for _, never := range []string{"SECRETVIN123", "44.1234", "-93.6543"} {
		if strings.Contains(out.Text, never) {
			t.Errorf("must not contain %q:\n%s", never, out.Text)
		}
	}
	// A viewer cannot read it.
	do(t, admin, "POST", ts.URL+"/api/users", `{"name":"fam","password":"viewer-password-1","role":"viewer"}`, xh)
	vj, _ := cookieJar()
	v := &http.Client{Jar: vj}
	do(t, v, "POST", ts.URL+"/api/login", `{"name":"fam","password":"viewer-password-1"}`, nil)
	if code, _ := do(t, v, "GET", ts.URL+"/api/debug-report", "", nil); code != 403 {
		t.Fatalf("viewer: %d", code)
	}
}
