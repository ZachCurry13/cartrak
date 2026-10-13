package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

func TestTripsCanBeCombinedIntoAJourneyWithTotalsAndUngrouped(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	base := time.Now().Add(-48 * time.Hour)
	mk := func(startH float64, km float64) int64 {
		start := base.Add(time.Duration(startH * float64(time.Hour)))
		end := start.Add(time.Hour).Unix()
		a, b := 100.0, 100.0+km
		tr := &store.Trip{StartTS: start.Unix(), EndTS: &end, DistanceKm: km, StartOdo: &a, EndOdo: &b, Category: "work", TZ: "UTC"}
		if err := srv.S.AddManualTrip(tr); err != nil {
			t.Fatal(err)
		}
		return tr.ID
	}
	a, b, c := mk(0, 100), mk(3, 50), mk(9, 20)
	post := func(body string) (int, string) { return do(t, admin, "POST", ts.URL+"/api/journeys", body, xh) }

	if code, _ := post(`{"name":"x","tripIds":[` + fmt.Sprint(a) + `]}`); code != 400 {
		t.Fatalf("one trip is not a journey: %d", code)
	}
	code, body := post(fmt.Sprintf(`{"name":"  Work trip  ","tripIds":[%d,%d]}`, a, b))
	var made struct{ ID int64 }
	if code != 200 || json.Unmarshal([]byte(body), &made) != nil || made.ID == 0 {
		t.Fatalf("create: %d %s", code, body)
	}
	if code, _ := post(fmt.Sprintf(`{"tripIds":[%d,%d]}`, b, c)); code != http.StatusConflict {
		t.Fatalf("a trip already in a journey: %d", code)
	}

	get := func() (j struct {
		Journey struct {
			Name       string
			Trips      int
			DistanceKm float64
		}
		Trips   []struct{ ID int64 }
		Summary struct {
			Trips      int
			DistanceKm float64
			MovingSec  int64
			ElapsedSec int64
			Stops      []struct{ Seconds int64 }
		}
	}) {
		code, b := do(t, admin, "GET", fmt.Sprintf("%s/api/journeys/%d", ts.URL, made.ID), "", nil)
		if code != 200 || json.Unmarshal([]byte(b), &j) != nil {
			t.Fatalf("detail: %d %s", code, b)
		}
		return
	}
	j := get()
	if j.Journey.Name != "Work trip" || j.Summary.Trips != 2 || j.Summary.DistanceKm != 150 || j.Summary.MovingSec != 7200 || j.Summary.ElapsedSec != 4*3600 || len(j.Summary.Stops) != 1 || j.Summary.Stops[0].Seconds != 2*3600 {
		t.Fatalf("%+v", j)
	}

	if code, b := do(t, admin, "PATCH", fmt.Sprintf("%s/api/journeys/%d", ts.URL, made.ID), fmt.Sprintf(`{"name":"Road trip","add":[%d],"remove":[%d]}`, c, a), xh); code != 200 {
		t.Fatalf("patch: %d %s", code, b)
	}
	if j := get(); j.Journey.Name != "Road trip" || j.Summary.Trips != 2 || j.Summary.DistanceKm != 70 {
		t.Fatalf("after editing: %+v", j)
	}
	if code, _ := do(t, admin, "GET", fmt.Sprintf("%s/api/trips/routes?journey=%d", ts.URL, made.ID), "", nil); code != 200 {
		t.Fatalf("routes: %d", code)
	}
	if code, _ := do(t, &http.Client{}, "GET", ts.URL+"/api/journeys", "", nil); code != 401 {
		t.Fatalf("anonymous: %d", code)
	}
	if code, _ := do(t, admin, "DELETE", fmt.Sprintf("%s/api/journeys/%d", ts.URL, made.ID), "", xh); code != 200 {
		t.Fatal("delete")
	}
	if tr, _ := srv.S.GetTrip(c); tr == nil || tr.JourneyID != nil {
		t.Fatalf("the trips stay, ungrouped: %+v", tr)
	}
}
