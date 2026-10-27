package server

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

func TestSettingTheOutingRelabelsItsTripsAndOnlyAdmins(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	if err := srv.S.SaveNamedPlace(&store.NamedPlace{Name: "Home", Lat: 45, Lon: -93, RadiusM: 150}); err != nil {
		t.Fatal(err)
	}
	t0 := time.Now().Add(-3 * time.Hour).Truncate(time.Minute)
	set := func(start time.Time, minutes int, sLat, sLon, eLat, eLon float64) *store.Trip {
		tr := &store.Trip{StartTS: start.Unix(), StartLat: &sLat, StartLon: &sLon, Category: "personal", CategorySource: "default"}
		if err := srv.S.CreateTrip(tr); err != nil {
			t.Fatal(err)
		}
		end := start.Add(time.Duration(minutes) * time.Minute).Unix()
		tr.EndTS, tr.EndLat, tr.EndLon, tr.DistanceKm, tr.Open = &end, &eLat, &eLon, 10, false
		if err := srv.S.UpdateTrip(tr); err != nil {
			t.Fatal(err)
		}
		return tr
	}
	set(t0, 30, 45, -93, 45, -93)                              // an earlier outing, back Home: not part of this one
	a := set(t0.Add(60*time.Minute), 20, 45, -93, 45.1, -93.2) // leaves Home
	b := set(t0.Add(100*time.Minute), 20, 45.1, -93.2, 45.2, -93.3)

	if code, _ := do(t, admin, "POST", ts.URL+"/api/outing", `{"category":"nonsense"}`, xh); code != 400 {
		t.Fatalf("an unknown category is refused: %d", code)
	}
	code, body := do(t, admin, "POST", ts.URL+"/api/outing", `{"category":"work"}`, xh)
	var out struct {
		Relabeled int `json:"relabeled"`
	}
	if err := json.Unmarshal([]byte(body), &out); err != nil || code != 200 || out.Relabeled != 2 {
		t.Fatalf("the two trips of this outing: %d %s", code, body)
	}
	for _, tr := range []*store.Trip{a, b} {
		got, _ := srv.S.GetTrip(tr.ID)
		if got.Category != "work" || got.CategorySource != "manual" {
			t.Fatalf("relabeled by hand: %+v", got)
		}
	}
	all, _ := srv.S.ListTrips(store.TripFilter{})
	if all[2].Category != "personal" {
		t.Fatalf("the earlier outing is untouched: %q", all[2].Category)
	}
	if _, body := do(t, admin, "GET", ts.URL+"/api/outing", "", nil); body == "" {
		t.Fatal("the outing can be read")
	}
	if code, _ := do(t, admin, "POST", ts.URL+"/api/outing", `{"category":""}`, xh); code != 200 {
		t.Fatalf("clearing: %d", code)
	}
}
