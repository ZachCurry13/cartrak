package server

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/geocode"
	"github.com/ZachCurry13/cartrak/internal/store"
)

func TestTripsListCarriesTheCityTheyStartedAndEndedIn(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	f := func(x float64) *float64 { return &x }
	tr := finishedTrip(t, srv, time.Now().Add(-48*time.Hour), 12, func(tr *store.Trip) {
		tr.StartLat, tr.StartLon, tr.EndLat, tr.EndLon = f(42.88), f(-78.87), f(43.1), f(-79.05)
	})
	_ = srv.S.SetPlace(geocode.Key(42.88, -78.87), "Buffalo")
	_ = srv.S.SetPlace(geocode.Key(43.1, -79.05), "Niagara Falls")

	_, body := do(t, admin, "GET", ts.URL+"/api/trips", "", nil)
	var list []struct {
		ID        int64  `json:"id"`
		StartCity string `json:"startCity"`
		EndCity   string `json:"endCity"`
	}
	if err := json.Unmarshal([]byte(body), &list); err != nil {
		t.Fatal(err)
	}
	for _, x := range list {
		if x.ID == tr.ID {
			if x.StartCity != "Buffalo" || x.EndCity != "Niagara Falls" {
				t.Fatalf("cities: %+v", x)
			}
			return
		}
	}
	t.Fatal("trip missing from the list")
}
