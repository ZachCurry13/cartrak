package server

import (
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/settings"
	"github.com/ZachCurry13/cartrak/internal/store"
)

// A trip whose road cannot be estimated can be dismissed: it stays a straight line and is no longer offered.
func TestADismissedTripIsNoLongerWaitingForARoadEstimate(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	srv.Cfg.Set(settings.RouteKey, "on")
	start := time.Now().Add(-48 * time.Hour)
	tr := &store.Trip{StartTS: start.Unix(), Category: "work", CategorySource: "default"}
	if err := srv.S.CreateTrip(tr); err != nil {
		t.Fatal(err)
	}
	end := start.Add(20 * time.Minute).Unix()
	tr.EndTS, tr.DistanceKm, tr.Open = &end, 12, false
	if err := srv.S.UpdateTrip(tr); err != nil {
		t.Fatal(err)
	}
	pending := func() int {
		_, body := do(t, admin, "GET", ts.URL+"/api/trips/estimate-routes", "", nil)
		var v struct {
			Pending int `json:"pending"`
		}
		_ = json.Unmarshal([]byte(body), &v)
		return v.Pending
	}
	if pending() != 1 {
		t.Fatalf("one trip is waiting")
	}
	if code, _ := do(t, admin, "POST", ts.URL+"/api/trips/no-route", `{"ids":[]}`, xh); code != 400 {
		t.Fatalf("nothing to dismiss is refused: %d", code)
	}
	code, body := do(t, admin, "POST", ts.URL+"/api/trips/no-route", `{"ids":[`+strconv.FormatInt(tr.ID, 10)+`,99999]}`, xh)
	if code != 200 {
		t.Fatalf("dismiss: %d %s", code, body)
	}
	if pending() != 0 {
		t.Fatal("it is not offered again")
	}
}
