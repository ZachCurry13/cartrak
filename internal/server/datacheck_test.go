package server

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

func TestDataCheckFlagsAnImpossibleTripAndStaysQuietOtherwise(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	get := func() (n int, kinds []string) {
		code, b := do(t, admin, "GET", ts.URL+"/api/datacheck", "", nil)
		var out struct {
			Findings []struct{ Kind string }
		}
		if code != 200 || json.Unmarshal([]byte(b), &out) != nil {
			t.Fatalf("%d %s", code, b)
		}
		for _, f := range out.Findings {
			kinds = append(kinds, f.Kind)
		}
		return len(out.Findings), kinds
	}
	if n, _ := get(); n != 0 {
		t.Fatal("an empty database has nothing wrong")
	}
	start := time.Now().Add(-3 * time.Hour)
	end := start.Add(10 * time.Minute).Unix()
	tr := &store.Trip{StartTS: start.Unix(), EndTS: &end, DistanceKm: 120, Open: false, Category: "personal", CategorySource: "default"}
	if err := srv.S.CreateTrip(tr); err != nil {
		t.Fatal(err)
	}
	tr.EndTS, tr.Open = &end, false
	if err := srv.S.UpdateTrip(tr); err != nil {
		t.Fatal(err)
	}
	if n, k := get(); n != 1 || k[0] != "fast" {
		t.Fatalf("720 km/h: %d %v", n, k)
	}
}
