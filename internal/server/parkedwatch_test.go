package server

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/ZachCurry13/cartrak/internal/store"
)

func TestTheParkedWatchPageShowsTotalsAndEventsAndCanBeReset(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	srv.S.NoteParkedReading(0.3, 100)
	srv.S.NoteParkedReading(1.8, 200)
	srv.S.AddParkedEvent(store.ParkedEvent{TS: 200, Kind: "motion", Value: 1.8, Note: "1.80 m/s²"})
	_, body := do(t, admin, "GET", ts.URL+"/api/parked-watch", "", nil)
	var d struct {
		On    bool
		Stats struct {
			Checked int
			MaxSeen float64
		}
		Events []struct{ Kind string }
	}
	if err := json.Unmarshal([]byte(body), &d); err != nil || d.Stats.Checked != 2 || d.Stats.MaxSeen != 1.8 || len(d.Events) != 1 {
		t.Fatalf("%s %v", body, err)
	}
	if code, _ := do(t, admin, "POST", ts.URL+"/api/parked-watch/reset", "", xh); code != 200 {
		t.Fatal("reset")
	}
	_, body = do(t, admin, "GET", ts.URL+"/api/parked-watch", "", nil)
	json.Unmarshal([]byte(body), &d)
	if d.Stats.Checked != 0 || len(d.Events) != 0 {
		t.Fatalf("a fresh trial: %s", body)
	}
	if c, _ := do(t, &http.Client{}, "GET", ts.URL+"/api/parked-watch", "", nil); c != 401 {
		t.Fatalf("anonymous: %d", c)
	}
}
