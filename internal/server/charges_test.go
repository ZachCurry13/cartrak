package server

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

func TestChargingSessionsShowEnergyAndTheOwnerCanMarkHome(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	lat, lon := 40.0, -75.0
	start := time.Now().Add(-5 * time.Hour)
	mk := func(at time.Time, from, to float64, la float64) int64 {
		c := &store.Charge{StartTS: at.Unix(), Lat: &la, Lon: &lon, StartKwh: &from, PeakKw: 7}
		if err := srv.S.CreateCharge(c); err != nil {
			t.Fatal(err)
		}
		end := at.Add(3 * time.Hour).Unix()
		c.EndTS, c.EndKwh, c.Open = &end, &to, false
		if err := srv.S.UpdateCharge(c); err != nil {
			t.Fatal(err)
		}
		return c.ID
	}
	a := mk(start, 40, 58, lat)
	mk(start.Add(-48*time.Hour), 30, 36, 41.0)

	type sess struct {
		ID       int64    `json:"id"`
		AddedKwh *float64 `json:"addedKwh"`
		Where    string   `json:"where"`
		Kind     string   `json:"kind"`
	}
	list := func() []sess {
		code, b := do(t, admin, "GET", ts.URL+"/api/charges", "", nil)
		var out []sess
		if code != 200 || json.Unmarshal([]byte(b), &out) != nil {
			t.Fatalf("%d %s", code, b)
		}
		return out
	}
	got := list()
	if len(got) != 2 || got[0].AddedKwh == nil || math.Abs(*got[0].AddedKwh-18) > 1e-6 || got[0].Where != "" {
		t.Fatalf("before home is known the place is unknown: %+v", got)
	}
	if code, b := do(t, admin, "PATCH", fmt.Sprintf("%s/api/charges/%d", ts.URL, a), `{"kind":"home"}`, xh); code != 200 {
		t.Fatalf("mark home: %d %s", code, b)
	}
	got = list()
	if got[0].Where != "home" || got[0].Kind != "home" || got[1].Where != "public" {
		t.Fatalf("the other session, a degree away, is public: %+v", got)
	}
	if code, _ := do(t, admin, "PATCH", fmt.Sprintf("%s/api/charges/%d", ts.URL, a), `{"kind":"boat"}`, xh); code != 400 {
		t.Fatalf("bad kind: %d", code)
	}
	code, b := do(t, admin, "GET", ts.URL+"/api/charges/summary", "", nil)
	var sum struct {
		Months []struct {
			Sessions int     `json:"sessions"`
			Kwh      float64 `json:"kwh"`
		}
		Currency string
		PriceSet bool
	}
	if code != 200 || json.Unmarshal([]byte(b), &sum) != nil || len(sum.Months) == 0 || sum.Currency == "" || sum.PriceSet {
		t.Fatalf("summary: %d %s", code, b)
	}
	if c, _ := do(t, &http.Client{}, "GET", ts.URL+"/api/charges/summary", "", nil); c != 401 {
		t.Fatalf("anonymous: %d", c)
	}
}
