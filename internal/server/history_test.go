package server

import (
	"encoding/json"
	"testing"
	"time"
)

func TestHistoryAndStatsAnswerForThisMonth(t *testing.T) {
	ts, _ := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	month := time.Now().Format("2006-01")
	code, b := do(t, admin, "GET", ts.URL+"/api/history?month="+month, "", nil)
	var h struct {
		Month  string
		Events []map[string]any
		Totals map[string]any
	}
	if code != 200 || json.Unmarshal([]byte(b), &h) != nil || h.Month != month || h.Events == nil || h.Totals == nil {
		t.Fatalf("history %d %s", code, b)
	}
	if code, _ := do(t, admin, "GET", ts.URL+"/api/history?month=nope", "", nil); code != 400 {
		t.Fatalf("bad month: %d", code)
	}
	for _, p := range []string{"day", "month", "year"} {
		code, b = do(t, admin, "GET", ts.URL+"/api/stats?period="+p, "", nil)
		var s struct {
			Stats struct{ Buckets []map[string]any }
		}
		if code != 200 || json.Unmarshal([]byte(b), &s) != nil || len(s.Stats.Buckets) == 0 {
			t.Fatalf("stats %s %d %s", p, code, b)
		}
	}
	if code, _ := do(t, admin, "GET", ts.URL+"/api/stats?period=week", "", nil); code != 400 {
		t.Fatalf("bad period: %d", code)
	}
}
