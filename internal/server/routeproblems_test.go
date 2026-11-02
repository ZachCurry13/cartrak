package server

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

func TestTripsWithoutARoadStayListedUntilFixedOrDismissed(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	mk := func(minsAgo int) *store.Trip {
		start := time.Now().Add(-time.Duration(minsAgo) * time.Minute)
		end := start.Add(10 * time.Minute).Unix()
		tr := &store.Trip{StartTS: start.Unix(), Category: "personal", CategorySource: "default"}
		if err := srv.S.CreateTrip(tr); err != nil {
			t.Fatal(err)
		}
		tr.EndTS, tr.Open, tr.DistanceKm = &end, false, 5
		srv.S.UpdateTrip(tr)
		return tr
	}
	a, b, c, old := mk(300), mk(200), mk(100), mk(50)
	// a: gave up with a reason; b: gave up with a road that did not fit; c: gave up before reasons were kept; old is fine
	_ = srv.S.SaveNoRoute(a.ID)
	_ = srv.S.NoteRouteReview(a.ID, "the car reported too few positions to route between", 0, 5)
	_ = srv.S.SaveNoRoute(b.ID)
	_ = srv.S.NoteRouteReview(b.ID, "", 12, 5)
	_ = srv.S.SaveNoRoute(c.ID)
	_ = srv.S.SaveRouteEstimate(old.ID, [][2]float64{{1, 1}, {2, 2}}, 5)

	list := func() []struct {
		ID     int64
		Why    string
		RoadKm float64
	} {
		code, body := do(t, admin, "GET", ts.URL+"/api/trips/route-problems", "", nil)
		var out struct {
			Items []struct {
				ID     int64
				Why    string
				RoadKm float64
			}
		}
		if code != 200 || json.Unmarshal([]byte(body), &out) != nil {
			t.Fatalf("%d %s", code, body)
		}
		return out.Items
	}
	got := list()
	if len(got) != 3 || got[0].ID != c.ID || got[1].ID != b.ID || got[2].ID != a.ID {
		t.Fatalf("all three, newest first, and not the trip that has a road: %+v", got)
	}
	if got[2].Why != "the car reported too few positions to route between" || got[1].RoadKm != 12 || got[1].Why == "" || got[0].Why == "" {
		t.Fatalf("each says why: %+v", got)
	}
	// asking again, as coming back to the page does, still lists them
	if len(list()) != 3 {
		t.Fatal("the list is kept, not lost after one visit")
	}
	if code, _ := do(t, admin, "POST", ts.URL+"/api/trips/no-route", fmt.Sprintf(`{"ids":[%d]}`, a.ID), xh); code != 200 {
		t.Fatalf("dismiss: %d", code)
	}
	_ = srv.S.SaveRouteEstimate(b.ID, [][2]float64{{1, 1}, {2, 2}}, 5) // a chosen road
	rest := list()
	if len(rest) != 1 || rest[0].ID != c.ID {
		t.Fatalf("a dismissed trip and a trip with a road are gone, the other stays: %+v", rest)
	}
}
