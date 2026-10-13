package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

func TestNamedPlacesLabelTripsAndCanBeMadeFromATrip(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	a, b := 40.0, 40.1
	lon := -75.0
	start := time.Now().Add(-2 * time.Hour)
	end := start.Add(30 * time.Minute).Unix()
	o1, o2 := 100.0, 120.0
	tr := &store.Trip{StartTS: start.Unix(), EndTS: &end, StartLat: &a, StartLon: &lon, EndLat: &b, EndLon: &lon, StartOdo: &o1, EndOdo: &o2, DistanceKm: 20, Category: "work", TZ: "UTC"}
	if err := srv.S.AddManualTrip(tr); err != nil {
		t.Fatal(err)
	}
	post := func(body string) int { c, _ := do(t, admin, "POST", ts.URL+"/api/named-places", body, xh); return c }
	if post(`{"name":""}`) != 400 || post(`{"name":"x","radiusM":5}`) != 400 || post(`{"name":"Nowhere"}`) != 400 {
		t.Fatal("bad names, sizes, and places that cannot be located are refused")
	}
	if post(fmt.Sprintf(`{"name":"Home","tripId":%d,"end":"start"}`, tr.ID)) != 200 || post(fmt.Sprintf(`{"name":"Office","tripId":%d,"end":"end","radiusM":300}`, tr.ID)) != 200 {
		t.Fatal("naming a trip's start and end")
	}
	_, body := do(t, admin, "GET", fmt.Sprintf("%s/api/trips/%d", ts.URL, tr.ID), "", nil)
	var d struct {
		Trip struct{ StartPlace, EndPlace string }
	}
	json.Unmarshal([]byte(body), &d)
	if d.Trip.StartPlace != "Home" || d.Trip.EndPlace != "Office" {
		t.Fatalf("the trip carries its places: %s", body)
	}
	_, list := do(t, admin, "GET", ts.URL+"/api/trips", "", nil)
	var rows []struct{ StartPlace, EndPlace string }
	json.Unmarshal([]byte(list), &rows)
	if len(rows) != 1 || rows[0].StartPlace != "Home" {
		t.Fatalf("and so does the list: %s", list)
	}
	_, pl := do(t, admin, "GET", ts.URL+"/api/named-places", "", nil)
	var places []struct {
		ID      int64
		Name    string
		RadiusM float64
	}
	json.Unmarshal([]byte(pl), &places)
	if len(places) != 2 {
		t.Fatalf("%s", pl)
	}
	if c, _ := do(t, admin, "PATCH", fmt.Sprintf("%s/api/named-places/%d", ts.URL, places[0].ID), `{"name":"Garage"}`, xh); c != 200 {
		t.Fatal("rename")
	}
	if c, _ := do(t, &http.Client{}, "GET", ts.URL+"/api/named-places", "", nil); c != 401 {
		t.Fatalf("anonymous: %d", c)
	}
	if c, _ := do(t, admin, "DELETE", fmt.Sprintf("%s/api/named-places/%d", ts.URL, places[1].ID), "", xh); c != 200 {
		t.Fatal("delete")
	}
}
