package server

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/store"
)

func TestAPhoneTrackThatStartedLateIsCompletedWithADottedGap(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	loc := srv.defaultLoc()
	day := time.Date(2026, 10, 18, 0, 0, 0, 0, loc)
	at := func(h, m int) time.Time { return day.Add(time.Duration(h)*time.Hour + time.Duration(m)*time.Minute) }
	fp := func(v float64) *float64 { return &v }
	a, b := [2]float64{44.0, -85.0}, [2]float64{44.5, -85.5}
	for i := 0; i <= 6; i++ { // the car's own positions all the way
		f := float64(i) / 6
		st := &ford.State{TS: at(7, 0).Add(time.Duration(i) * 15 * time.Minute), Lat: fp(a[0] + (b[0]-a[0])*f), Lon: fp(a[1] + (b[1]-a[1])*f), OdometerKm: fp(1000), Raw: []byte(`{}`)}
		if _, err := srv.S.SaveState("t", st); err != nil {
			t.Fatal(err)
		}
	}
	tr := &store.Trip{StartTS: at(7, 0).Unix(), StartLat: fp(a[0]), StartLon: fp(a[1]), Category: "personal", CategorySource: "default"}
	if err := srv.S.CreateTrip(tr); err != nil {
		t.Fatal(err)
	}
	e := at(8, 30).Unix()
	tr.EndTS, tr.EndLat, tr.EndLon, tr.DistanceKm, tr.Open = &e, fp(b[0]), fp(b[1]), 80, false
	if err := srv.S.UpdateTrip(tr); err != nil {
		t.Fatal(err)
	}
	var pts []store.Point // the phone's GPS came on for the last third and ended at the destination
	for i := 0; i <= 10; i++ {
		f := 0.66 + 0.34*float64(i)/10
		pts = append(pts, store.Point{TS: at(8, 0).Unix() + int64(i)*180, Lat: a[0] + (b[0]-a[0])*f, Lon: a[1] + (b[1]-a[1])*f})
	}
	if err := srv.S.SavePoints(1, pts); err != nil {
		t.Fatal(err)
	}
	_, body := do(t, admin, "GET", ts.URL+"/api/day?date=2026-10-18", "", nil)
	var out struct {
		Routes []struct {
			TripID int64
			Kind   string
			Points [][]float64
		}
	}
	if json.Unmarshal([]byte(body), &out) != nil || len(out.Routes) != 2 || out.Routes[0].Kind != "estimate" || out.Routes[1].Kind != "phone" || out.Routes[0].TripID != out.Routes[1].TripID {
		t.Fatalf("want a dotted gap then the phone track for one trip: %s", body)
	}
	if first := out.Routes[0].Points[0]; first[0] != a[0] || first[1] != a[1] {
		t.Fatalf("the gap starts where the trip did: %v", first)
	}
}
