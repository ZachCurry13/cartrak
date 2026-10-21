package server

import (
	"strings"
	"testing"
)

func TestBirthdayAndHolidayTouchesFollowThePerson(t *testing.T) {
	ts, _ := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	if _, body := do(t, admin, "GET", ts.URL+"/api/me/prefs", "", nil); !strings.Contains(body, `"birthday":""`) || !strings.Contains(body, `"holidayOff":false`) {
		t.Fatalf("nothing set: holiday touches on, no birthday: %s", body)
	}
	if code, _ := do(t, admin, "PUT", ts.URL+"/api/me/prefs", `{"birthday":"07-14","holidayOff":true}`, xh); code != 200 {
		t.Fatal("save")
	}
	if _, body := do(t, admin, "GET", ts.URL+"/api/me", "", nil); !strings.Contains(body, `"birthday":"07-14"`) {
		t.Fatalf("me carries the birthday for the theme: %s", body)
	}
	if _, body := do(t, admin, "GET", ts.URL+"/api/users", "", nil); strings.Contains(body, "07-14") {
		t.Fatalf("the people list never shows a birthday: %s", body)
	}
	for _, bad := range []string{`{"birthday":"13-01"}`, `{"birthday":"02-30"}`, `{"birthday":"1990-07-14"}`, `{"birthday":"7-14"}`} {
		if code, _ := do(t, admin, "PUT", ts.URL+"/api/me/prefs", bad, xh); code != 400 {
			t.Fatalf("%s should be refused", bad)
		}
	}
}
