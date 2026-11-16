package server

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/routeest"
	"github.com/ZachCurry13/cartrak/internal/store"
)

// A 25 km drive with the phone asleep for most of it: three fixes. That is no track, so the road is estimated (through those
// fixes), and the trip page offers to choose it.
func TestThreePhoneFixesOnALongDriveAreNotATrack(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	t0 := time.Date(2026, 11, 8, 10, 17, 0, 0, time.UTC)
	fp := func(v float64) *float64 { return &v }
	for i := 0; i < 20; i++ { // the car repeats where it started for the whole drive
		if _, err := srv.S.SaveState("t", &ford.State{TS: t0.Add(time.Duration(i) * 90 * time.Second), Lat: fp(45.20), Lon: fp(-93.66), Raw: []byte(`{}`)}); err != nil {
			t.Fatal(err)
		}
	}
	tr := &store.Trip{StartTS: t0.Unix(), StartLat: fp(45.20), StartLon: fp(-93.66), Category: "personal", CategorySource: "default"}
	if err := srv.S.CreateTrip(tr); err != nil {
		t.Fatal(err)
	}
	e := t0.Add(31 * time.Minute).Unix()
	tr.EndTS, tr.EndLat, tr.EndLon, tr.DistanceKm, tr.Open = &e, fp(45.30), fp(-93.55), 25, false
	if err := srv.S.UpdateTrip(tr); err != nil {
		t.Fatal(err)
	}
	pts := []store.Point{{TS: t0.Unix() + 60, Lat: 45.21, Lon: -93.65}, {TS: t0.Unix() + 900, Lat: 45.25, Lon: -93.60}, {TS: t0.Unix() + 1700, Lat: 45.29, Lon: -93.56}}
	if err := srv.S.SavePoints(1, pts); err != nil {
		t.Fatal(err)
	}
	_, body := do(t, admin, "GET", ts.URL+"/api/trips/1", "", nil)
	var out struct {
		PhoneFull bool
		Phone     []any
	}
	if json.Unmarshal([]byte(body), &out) != nil || out.PhoneFull || len(out.Phone) != 3 {
		t.Fatalf("three fixes are kept but are not a track: %s", body)
	}
	got, _ := srv.S.GetTrip(1)
	if srv.hasPhoneTrack(got) {
		t.Fatal("the road estimate must not skip this trip")
	}
	want := []routeest.Point{{45.20, -93.66}, {45.21, -93.65}, {45.25, -93.60}, {45.29, -93.56}, {45.30, -93.55}}
	rec := srv.recordedPoints(got)
	if len(rec) != len(want) {
		t.Fatalf("the start, the phone fixes in order and the end, without the car's repeats: %v", rec)
	}
	for i := range want {
		if rec[i] != want[i] {
			t.Fatalf("point %d: %v, want %v", i, rec[i], want[i])
		}
	}
	if !phoneFull([]store.Point{{Lat: 45.2, Lon: -93.6}, {Lat: 45.21, Lon: -93.61}, {Lat: 45.22, Lon: -93.62}}) {
		t.Fatal("fixes a kilometer or two apart are a track")
	}
}
