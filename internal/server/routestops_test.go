package server

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/routeest"
	"github.com/ZachCurry13/cartrak/internal/store"
)

func TestAStopGoesWhereItAddsTheLeastDetour(t *testing.T) {
	a, b, c := routeest.Point{45.0, -93.0}, routeest.Point{45.0, -92.0}, routeest.Point{45.0, -91.0}
	got := withStops([]routeest.Point{a, b, c}, []store.Via{{Name: "near the second leg", Lat: 45.01, Lon: -91.5}})
	if len(got) != 4 || got[2][1] != -91.5 {
		t.Fatalf("between b and c: %v", got)
	}
	two := withStops([]routeest.Point{a, c}, []store.Via{{Lat: 45.3, Lon: -92.0}})
	if len(two) != 3 || two[1][0] != 45.3 {
		t.Fatalf("start, stop, end: %v", two)
	}
}

func TestTheStopsOfATripAreKeptValidatedAndUsedByTheEstimate(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	fp := func(v float64) *float64 { return &v }
	start := time.Now().Add(-3 * time.Hour).Truncate(time.Second)
	end := start.Add(time.Hour).Unix()
	tr := &store.Trip{StartTS: start.Unix(), StartLat: fp(45.0), StartLon: fp(-93.0), Category: "personal", CategorySource: "default"}
	if err := srv.S.CreateTrip(tr); err != nil {
		t.Fatal(err)
	}
	tr.EndTS, tr.EndLat, tr.EndLon, tr.Open = &end, fp(45.0), fp(-91.0), false
	srv.S.UpdateTrip(tr)
	url := fmt.Sprintf("%s/api/trips/%d/route-stops", ts.URL, tr.ID)
	if code, b := do(t, admin, "POST", url, `{"stops":[{"name":"  A town  ","lat":45.4,"lon":-92.0}]}`, xh); code != 200 {
		t.Fatalf("%d %s", code, b)
	}
	code, b := do(t, admin, "GET", url, "", nil)
	var out struct{ Stops []store.Via }
	if code != 200 || json.Unmarshal([]byte(b), &out) != nil || len(out.Stops) != 1 || out.Stops[0].Name != "A town" {
		t.Fatalf("the stop is kept, trimmed: %d %s", code, b)
	}
	pts := srv.recordedPoints(tr)
	if len(pts) != 3 || pts[1][0] != 45.4 {
		t.Fatalf("the estimate goes start, stop, end: %v", pts)
	}
	if code, _ := do(t, admin, "POST", url, `{"stops":[{"lat":0,"lon":0}]}`, xh); code != 400 {
		t.Fatalf("a stop with no position is refused: %d", code)
	}
	six := strings.Repeat(`{"lat":45.4,"lon":-92},`, 6)
	if code, _ := do(t, admin, "POST", url, `{"stops":[`+strings.TrimSuffix(six, ",")+`]}`, xh); code != 400 {
		t.Fatalf("more than five stops is refused: %d", code)
	}
	if code, _ := do(t, admin, "POST", url, `{"stops":[]}`, xh); code != 200 {
		t.Fatal("stops can be cleared")
	}
	if pts := srv.recordedPoints(tr); len(pts) != 2 {
		t.Fatalf("back to start and end: %v", pts)
	}
}
