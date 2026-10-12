package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestAboutChangelogOnboarding(t *testing.T) {
	ts, _ := newTestServer(t)
	jar, _ := cookieJar()
	c := &http.Client{Jar: jar}
	x := map[string]string{"X-Requested-With": "cartrak"}

	if code, _ := do(t, c, "GET", ts.URL+"/api/about", "", nil); code != 401 {
		t.Fatalf("about without login = %d", code)
	}
	do(t, c, "POST", ts.URL+"/api/setup", `{"name":"owner","password":"correct-horse-battery"}`, nil)

	code, body := do(t, c, "GET", ts.URL+"/api/about", "", nil)
	if code != 200 || !strings.Contains(body, `"updateCheck":true`) {
		t.Fatalf("about: %d %s", code, body)
	}
	if code, body := do(t, c, "GET", ts.URL+"/api/changelog", "", nil); code != 200 || !strings.Contains(body, "# Changelog") {
		t.Fatalf("changelog: %d %.60s", code, body)
	}

	if _, body := do(t, c, "GET", ts.URL+"/api/onboarding", "", nil); !strings.Contains(body, `"dismissed":false`) || !strings.Contains(body, `"hasReading":false`) {
		t.Fatalf("onboarding: %s", body)
	}
	do(t, c, "POST", ts.URL+"/api/onboarding/dismiss", "", x)
	if _, body := do(t, c, "GET", ts.URL+"/api/onboarding", "", nil); !strings.Contains(body, `"dismissed":true`) {
		t.Fatalf("not dismissed: %s", body)
	}

	do(t, c, "POST", ts.URL+"/api/settings/update-check", `{"enabled":false}`, x)
	if _, body := do(t, c, "GET", ts.URL+"/api/about", "", nil); !strings.Contains(body, `"updateCheck":false`) {
		t.Fatalf("toggle failed: %s", body)
	}

	// Viewers may read but not change settings.
	do(t, c, "POST", ts.URL+"/api/users", `{"name":"fam","password":"viewer-password-1","role":"viewer"}`, x)
	vj, _ := cookieJar()
	v := &http.Client{Jar: vj}
	do(t, v, "POST", ts.URL+"/api/login", `{"name":"fam","password":"viewer-password-1"}`, nil)
	if code, _ := do(t, v, "GET", ts.URL+"/api/about", "", nil); code != 200 {
		t.Fatalf("viewer about = %d", code)
	}
	if code, _ := do(t, v, "POST", ts.URL+"/api/settings/update-check", `{"enabled":true}`, x); code != 403 {
		t.Fatalf("viewer changed settings = %d", code)
	}
	if code, _ := do(t, v, "POST", ts.URL+"/api/onboarding/dismiss", "", x); code != 403 {
		t.Fatalf("viewer dismissed onboarding = %d", code)
	}
}
