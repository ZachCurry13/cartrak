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

func TestTheChargersOwnNumberIsCheckedStoredAndLearnedFromAfterThree(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	lon := -75.0
	var ids []int64
	for i := 0; i < 3; i++ {
		la, from := 40.0, 40.0
		c := &store.Charge{StartTS: time.Now().Add(-time.Duration(i+1) * 24 * time.Hour).Unix(), Lat: &la, Lon: &lon, StartKwh: &from, PeakKw: 7}
		srv.S.CreateCharge(c)
		end, to := c.StartTS+3600, 58.0 // 18 kWh added
		c.EndTS, c.EndKwh, c.Open = &end, &to, false
		srv.S.UpdateCharge(c)
		ids = append(ids, c.ID)
	}
	patch := func(id int64, body string) int {
		c, _ := do(t, admin, "PATCH", fmt.Sprintf("%s/api/charges/%d", ts.URL, id), body, xh)
		return c
	}
	if patch(ids[0], `{"chargerKwh":10}`) != 400 || patch(ids[0], `{"chargerKwh":0}`) != 400 {
		t.Fatal("less than the battery got, or zero, is refused")
	}
	eff := func() (float64, string, int) {
		_, b := do(t, admin, "GET", ts.URL+"/api/charges/summary", "", nil)
		var out struct {
			Efficiency struct {
				Value   float64
				Source  string
				Samples int
			}
		}
		json.Unmarshal([]byte(b), &out)
		return out.Efficiency.Value, out.Efficiency.Source, out.Efficiency.Samples
	}
	if v, src, n := eff(); v != 0.9 || src != "setting" || n != 0 {
		t.Fatalf("before any entry: %v %s %d", v, src, n)
	}
	for i, id := range ids {
		if patch(id, `{"chargerKwh":21.2}`) != 200 {
			t.Fatalf("entry %d", i)
		}
		if i == 1 {
			if _, src, n := eff(); src != "setting" || n != 2 {
				t.Fatalf("two entries are not enough to trust: %s %d", src, n)
			}
		}
	}
	if v, src, n := eff(); src != "measured" || n != 3 || v < 0.84 || v > 0.86 {
		t.Fatalf("three entries: %v %s %d", v, src, n)
	}
	if patch(ids[0], `{"clearCharger":true}`) != 200 {
		t.Fatal("clear")
	}
	if _, src, _ := eff(); src != "setting" {
		t.Fatalf("back to the setting: %s", src)
	}
}
