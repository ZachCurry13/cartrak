package server

import (
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

func TestThinKeepsEndsAndSpreadsTheRest(t *testing.T) {
	var ps []store.Point
	for i := 0; i < 5000; i++ {
		ps = append(ps, store.Point{TS: int64(i), Lat: float64(i) / 100, Lon: 1})
	}
	out := thin(ps, 1000)
	if len(out) != 1000 || out[0].TS != 0 || out[len(out)-1].TS != 4999 {
		t.Fatalf("thinned to %d, first %d last %d", len(out), out[0].TS, out[len(out)-1].TS)
	}
	for i := 1; i < len(out); i++ {
		if out[i].TS <= out[i-1].TS {
			t.Fatalf("points out of order at %d", i)
		}
	}
	if short := thin(ps[:10], 1000); len(short) != 10 {
		t.Fatalf("a short route must be left alone: %d", len(short))
	}
}

func TestTripDetailIncludesThePhoneRoute(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	devID, _ := newDevice(t, ts.URL, admin, "Phone")
	now := time.Now().Unix()
	end := now - 600
	tr := &store.Trip{StartTS: now - 1800, EndTS: &end, Category: "personal", CategorySource: "default"}
	if err := srv.S.CreateTrip(tr); err != nil {
		t.Fatal(err)
	}
	tr.Open = false
	if err := srv.S.UpdateTrip(tr); err != nil {
		t.Fatal(err)
	}
	srv.S.SavePoints(devID, []store.Point{
		{TS: now - 4000, Lat: 9, Lon: 9}, // well before the trip
		{TS: now - 1790, Lat: 44.9, Lon: -93.2}, {TS: now - 1000, Lat: 44.95, Lon: -93.25},
		{TS: now - 100, Lat: 8, Lon: 8}, // after the trip
	})
	var out struct {
		Phone []store.Point `json:"phone"`
	}
	_, body := do(t, admin, "GET", ts.URL+"/api/trips/"+strconv.FormatInt(tr.ID, 10), "", nil)
	if err := json.Unmarshal([]byte(body), &out); err != nil || len(out.Phone) != 2 || out.Phone[0].Lat != 44.9 {
		t.Fatalf("phone route should hold only this trip's points: %s", body)
	}
	_, saver := do(t, admin, "GET", ts.URL+"/api/trips/"+strconv.FormatInt(tr.ID, 10)+"?samples=0", "", nil)
	var sv struct {
		Phone []store.Point `json:"phone"`
	}
	json.Unmarshal([]byte(saver), &sv)
	if len(sv.Phone) != 0 {
		t.Fatalf("data saver should skip the phone route too: %s", saver)
	}
}
