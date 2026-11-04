package server

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

func TestTireSetsAreKeptValidatedAndTheOneOnNowIsReported(t *testing.T) {
	ts, _ := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	url := ts.URL + "/api/tire-sets"
	now := time.Now()
	add := func(body string) (int, string) { return do(t, admin, "POST", url, body, xh) }
	if code, b := add(fmt.Sprintf(`{"name":"Summer","kind":"summer","fromTs":%d}`, now.AddDate(0, -5, 0).Unix())); code != 200 {
		t.Fatalf("%d %s", code, b)
	}
	if code, b := add(fmt.Sprintf(`{"name":"  Winter  ","kind":"winter","fromTs":%d,"note":"on the black wheels"}`, now.AddDate(0, 0, -2).Unix())); code != 200 {
		t.Fatalf("%d %s", code, b)
	}
	for _, bad := range []string{`{"name":"","kind":"winter","fromTs":1700000000}`, `{"name":"x","kind":"racing","fromTs":1700000000}`,
		fmt.Sprintf(`{"name":"x","kind":"winter","fromTs":%d}`, now.AddDate(0, 0, 30).Unix()), `{"name":"x","kind":"winter","fromTs":0}`} {
		if code, _ := add(bad); code != 400 {
			t.Errorf("%s should be refused: %d", bad, code)
		}
	}
	var out struct {
		Sets    []store.TireSet
		Current *store.TireSet
	}
	code, b := do(t, admin, "GET", url, "", nil)
	if code != 200 || json.Unmarshal([]byte(b), &out) != nil || len(out.Sets) != 2 || out.Current == nil || out.Current.Name != "Winter" || out.Sets[0].Name != "Summer" {
		t.Fatalf("two sets, oldest first, winter on now and trimmed: %d %s", code, b)
	}
	if code, _ := do(t, admin, "PATCH", fmt.Sprintf("%s/%d", url, out.Sets[0].ID), fmt.Sprintf(`{"name":"Summer set","kind":"allseason","fromTs":%d}`, out.Sets[0].FromTS), xh); code != 200 {
		t.Fatal("a set can be edited")
	}
	if code, _ := do(t, admin, "DELETE", fmt.Sprintf("%s/%d", url, out.Sets[1].ID), "", xh); code != 200 {
		t.Fatal("a set can be removed")
	}
	_, b = do(t, admin, "GET", url, "", nil)
	if json.Unmarshal([]byte(b), &out) != nil || len(out.Sets) != 1 || out.Sets[0].Kind != "allseason" || out.Current == nil || out.Current.Name != "Summer set" {
		t.Fatalf("removing the winter set puts the one before it back on: %s", b)
	}
}

func TestTheTireComparisonUsesOnlyTripsWhereBothTiresDrovePastTheSameTemperature(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	fp := func(v float64) *float64 { return &v }
	base := time.Now().AddDate(0, 0, -20).Truncate(time.Hour)
	srv.S.AddTireSet(store.TireSet{Name: "Summer", Kind: "summer", FromTS: base.AddDate(0, 0, -30).Unix()})
	srv.S.AddTireSet(store.TireSet{Name: "Winter", Kind: "winter", FromTS: base.Add(10 * 24 * time.Hour).Unix()})
	// one trip each day, 25 km in 30 minutes; the first ten days on summer tires, the rest on winter tires, both at 4 C
	for d := 0; d < 20; d++ {
		start := base.Add(time.Duration(d) * 24 * time.Hour)
		per := 0.18
		if d >= 10 {
			per = 0.18 * 1.07 // winter tires: 7% more
		}
		end := start.Add(30 * time.Minute).Unix()
		tr := &store.Trip{StartTS: start.Unix(), StartKwh: fp(80), Category: "personal", CategorySource: "default"}
		if err := srv.S.CreateTrip(tr); err != nil {
			t.Fatal(err)
		}
		tr.EndTS, tr.Open, tr.DistanceKm, tr.EndKwh = &end, false, 25, fp(80-25*per)
		srv.S.UpdateTrip(tr)
		srv.S.DB.Exec(`INSERT INTO trends(ts,outside_c) VALUES(?,?)`, start.Unix()+60, 4.0)
	}
	code, b := do(t, admin, "GET", ts.URL+"/api/tire-sets/compare", "", nil)
	var out struct {
		Result struct {
			Verdict string
			Pct     float64
			Matched int
		}
		HasWinter, HasOther bool
	}
	if code != 200 || json.Unmarshal([]byte(b), &out) != nil || out.Result.Verdict != "fair" || out.Result.Matched != 1 || out.Result.Pct < 6.9 || out.Result.Pct > 7.1 || !out.HasWinter || !out.HasOther {
		t.Fatalf("seven percent at the same temperature, 250 km on each (one cell: fair): %d %s", code, b)
	}
}
