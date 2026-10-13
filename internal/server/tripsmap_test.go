package server

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/store"
)

func TestOverviewRoutesUsePhoneTrackThenFordReadings(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	devID, _ := newDevice(t, ts.URL, admin, "Phone")
	now := time.Now().Unix()

	mk := func(start, end int64, cat string) *store.Trip {
		tr := &store.Trip{StartTS: start, EndTS: &end, Category: cat, CategorySource: "default"}
		if err := srv.S.CreateTrip(tr); err != nil {
			t.Fatal(err)
		}
		tr.Open = false
		srv.S.UpdateTrip(tr)
		return tr
	}
	withPhone := mk(now-7200, now-6600, "work")
	var phone []store.Point
	for i := 0; i < 200; i++ {
		phone = append(phone, store.Point{TS: now - 7190 + int64(i), Lat: 44 + float64(i)/1000, Lon: -93})
	}
	srv.S.SavePoints(devID, phone)

	fordOnly := mk(now-3600, now-3000, "personal")
	lat, lon := 45.0, -94.0
	for i := 0; i < 3; i++ {
		lat += 0.01
		srv.S.SaveState("test", sampleAt(time.Unix(now-3590+int64(i)*60, 0), lat, lon))
	}
	mk(now-1800, now-1700, "personal") // no GPS at all

	_, body := do(t, admin, "GET", ts.URL+"/api/trips/routes", "", nil)
	var out []routeOut
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 {
		t.Fatalf("a trip with no GPS has no route: %s", body)
	}
	byID := map[int64]routeOut{}
	for _, r := range out {
		byID[r.ID] = r
	}
	if p := byID[withPhone.ID]; len(p.Points) != overviewPoints || p.Points[0][0] != 44.0 || p.Category != "work" {
		t.Fatalf("200 phone points should be thinned to %d, first kept: %d", overviewPoints, len(p.Points))
	}
	if f := byID[fordOnly.ID]; len(f.Points) != 3 || f.Category != "personal" {
		t.Fatalf("Ford readings are used when the phone sent nothing: %+v", f)
	}
	_, onlyWork := do(t, admin, "GET", ts.URL+"/api/trips/routes?category=work", "", nil)
	if strings.Contains(onlyWork, `"personal"`) || !strings.Contains(onlyWork, `"work"`) {
		t.Fatalf("filters follow the trip list: %s", onlyWork)
	}
}

func sampleAt(at time.Time, lat, lon float64) *ford.State {
	return &ford.State{TS: at, Lat: &lat, Lon: &lon, Raw: []byte(`{}`)}
}
