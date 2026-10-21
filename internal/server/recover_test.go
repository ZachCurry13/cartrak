package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestABugAnswersWithAnErrorAndIsNotedForTheDebugReport(t *testing.T) {
	h := recoverer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var m map[string]int
		m["x"] = 1 // a nil map: the kind of bug that would drop the connection
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("POST", "/api/push/prefs", nil))
	if rec.Code != 500 || !strings.Contains(rec.Body.String(), "debug report") {
		t.Fatalf("an error the app can show: %d %s", rec.Code, rec.Body.String())
	}
	got := RecentServerErrors()
	if len(got) == 0 || !strings.Contains(got[len(got)-1], "POST /api/push/prefs") || !strings.Contains(got[len(got)-1], "nil map") {
		t.Fatalf("noted: %v", got)
	}
}
