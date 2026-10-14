package server

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/store"
)

func TestTimeUseSplitsTheWindowIntoDrivingChargingAndParked(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	now := time.Now()
	srv.S.SaveState("t", &ford.State{TS: now.Add(-5 * 24 * time.Hour), Raw: []byte(`{}`)}) // the first reading
	start := now.Add(-2 * 24 * time.Hour)
	end := start.Add(2 * time.Hour).Unix()
	tr := &store.Trip{StartTS: start.Unix(), EndTS: &end, Category: "work", TZ: "UTC"}
	if err := srv.S.AddManualTrip(tr); err != nil {
		t.Fatal(err)
	}
	c := &store.Charge{StartTS: start.Add(3 * time.Hour).Unix()}
	srv.S.CreateCharge(c)
	cend := start.Add(7 * time.Hour).Unix()
	c.EndTS, c.Open = &cend, false
	srv.S.UpdateCharge(c)

	code, body := do(t, admin, "GET", ts.URL+"/api/time-use?days=30", "", nil)
	var r struct {
		Driving, Charging, Parked int64
		Days                      []struct{ Date string }
		Places                    []struct{ Name string }
	}
	if code != 200 || json.Unmarshal([]byte(body), &r) != nil {
		t.Fatalf("%d %s", code, body)
	}
	if r.Driving != 7200 || r.Charging != 4*3600 || r.Parked < 4*24*3600 || len(r.Days) < 5 || len(r.Places) != 1 || r.Places[0].Name != "Elsewhere" {
		t.Fatalf("%s", body)
	}
	if c, _ := do(t, &http.Client{}, "GET", ts.URL+"/api/time-use", "", nil); c != 401 {
		t.Fatalf("anonymous: %d", c)
	}
}
