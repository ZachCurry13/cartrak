package server

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

func TestARemoteStartWhilePluggedInIsNotACharge(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	loc := srv.defaultLoc()
	day := time.Now().In(loc).AddDate(0, 0, -2)
	start := time.Date(day.Year(), day.Month(), day.Day(), 7, 40, 0, 0, loc)
	add := func(at time.Time, minutes int, from, to float64) int64 {
		c := &store.Charge{StartTS: at.Unix(), StartKwh: &from, PeakKw: 3.5}
		if err := srv.S.CreateCharge(c); err != nil {
			t.Fatal(err)
		}
		end := at.Add(time.Duration(minutes) * time.Minute).Unix()
		c.EndTS, c.EndKwh, c.Open = &end, &to, false
		if err := srv.S.UpdateCharge(c); err != nil {
			t.Fatal(err)
		}
		return c.ID
	}
	heating := add(start, 8, 64.35, 64.40)                     // 0.05 kWh: the climate control, during a remote start
	real := add(start.Add(8*time.Hour), 90, 40, 60)            // a real charge
	other := add(start.Add(-20*time.Hour), 10, 50, 50)         // a tiny session with no remote start near it
	if _, err := srv.S.RemoteStartsBetween(0, 1); err != nil { // makes the table
		t.Fatal(err)
	}
	if _, err := srv.S.DB.Exec(`INSERT INTO remote_starts(start_ts,end_ts,restarts,drove) VALUES(?,?,0,0)`, start.Unix()+30, start.Add(22*time.Minute).Unix()); err != nil {
		t.Fatal(err)
	}

	_, body := do(t, admin, "GET", ts.URL+"/api/charges", "", nil)
	var list []struct {
		ID     int64 `json:"id"`
		Remote bool  `json:"remote"`
	}
	if err := json.Unmarshal([]byte(body), &list); err != nil {
		t.Fatal(err)
	}
	got := map[int64]bool{}
	for _, c := range list {
		got[c.ID] = c.Remote
	}
	if !got[heating] || got[real] || got[other] {
		t.Fatalf("only the heating session is a remote start: heating=%v real=%v other=%v", got[heating], got[real], got[other])
	}

	_, body = do(t, admin, "GET", ts.URL+"/api/day?date="+start.Format("2006-01-02"), "", nil)
	var d struct {
		Events []struct {
			Kind string `json:"kind"`
			ID   int64  `json:"id"`
		} `json:"events"`
	}
	if err := json.Unmarshal([]byte(body), &d); err != nil {
		t.Fatal(err)
	}
	for _, e := range d.Events {
		if e.Kind == "charging" && e.ID == heating {
			t.Fatalf("the day does not list the heating session as a charge: %s", body)
		}
	}
	found := false
	for _, e := range d.Events {
		found = found || (e.Kind == "charging" && e.ID == real)
	}
	if !found {
		t.Fatalf("the real charge is still in the day: %s", body)
	}
}
