package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/routeest"
	"github.com/ZachCurry13/cartrak/internal/store"
)

func TestATripWithOnlyTwoCarPositionsGetsAnEstimatedRoadOnceAndOnlyWhenSwitchedOn(t *testing.T) {
	var calls int32
	osrm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.Write([]byte(`{"code":"Ok","routes":[{"distance":31000,"geometry":{"coordinates":[[-85.60,44.76],[-85.55,44.80],[-85.50,44.85]]}}]}`))
	}))
	defer osrm.Close()
	ts, srv := healthServer(t, fakeProv{}, false, false)
	srv.RouteEst = &routeest.Client{BaseURL: osrm.URL}
	admin := adminClient(t, ts.URL)
	t0 := time.Now().Add(-3 * time.Hour).Truncate(time.Second)
	fp := func(v float64) *float64 { return &v }
	for i, p := range [][2]float64{{44.76, -85.60}, {44.85, -85.50}} {
		st := &ford.State{TS: t0.Add(time.Duration(i*20) * time.Minute), Lat: fp(p[0]), Lon: fp(p[1]), OdometerKm: fp(1000 + float64(i)*30), Raw: []byte(`{}`)}
		if _, err := srv.S.SaveState("t", st); err != nil {
			t.Fatal(err)
		}
	}
	end := t0.Add(20 * time.Minute).Unix()
	tr := &store.Trip{StartTS: t0.Unix(), StartLat: fp(44.76), StartLon: fp(-85.60), Category: "personal", CategorySource: "default"}
	if err := srv.S.CreateTrip(tr); err != nil {
		t.Fatal(err)
	}
	tr.EndTS, tr.EndLat, tr.EndLon, tr.DistanceKm, tr.Open = &end, fp(44.85), fp(-85.50), 30, false
	if err := srv.S.UpdateTrip(tr); err != nil {
		t.Fatal(err)
	}
	url := fmt.Sprintf("%s/api/trips/%d", ts.URL, tr.ID)
	get := func() map[string]any {
		code, b := do(t, admin, "GET", url, "", nil)
		var out map[string]any
		if code != 200 || json.Unmarshal([]byte(b), &out) != nil {
			t.Fatalf("%d %s", code, b)
		}
		return out
	}
	if get()["estimate"] != nil || atomic.LoadInt32(&calls) != 0 {
		t.Fatal("nothing is sent while the lookup is off")
	}
	if err := srv.Cfg.KV.SetKV("setting:CARTRAK_ROUTE_LOOKUP", "on"); err != nil {
		t.Fatal(err)
	}
	est, _ := get()["estimate"].(map[string]any)
	if est == nil || len(est["points"].([]any)) != 3 || est["km"].(float64) != 31 {
		t.Fatalf("an estimate: %v", est)
	}
	get()
	if atomic.LoadInt32(&calls) != 1 {
		t.Fatalf("the service is asked once and the answer kept: %d calls", calls)
	}
}

func TestADriveTheOdometerCountedShortCanUseTheRoadsLengthOrBeTyped(t *testing.T) {
	osrm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"code":"Ok","routes":[{"distance":30000,"geometry":{"coordinates":[[-85.60,44.76],[-85.55,44.80],[-85.50,44.85]]}}]}`))
	}))
	defer osrm.Close()
	ts, srv := healthServer(t, fakeProv{}, false, false)
	srv.RouteEst = &routeest.Client{BaseURL: osrm.URL}
	admin := adminClient(t, ts.URL)
	_ = srv.Cfg.KV.SetKV("setting:CARTRAK_ROUTE_LOOKUP", "on")
	fp := func(v float64) *float64 { return &v }
	t0 := time.Now().Add(-3 * time.Hour).Truncate(time.Second)
	end := t0.Add(13 * time.Minute).Unix()
	tr := &store.Trip{StartTS: t0.Unix(), StartLat: fp(44.76), StartLon: fp(-85.60), StartOdo: fp(1000), Category: "personal", CategorySource: "default"}
	if err := srv.S.CreateTrip(tr); err != nil {
		t.Fatal(err)
	}
	tr.EndTS, tr.EndLat, tr.EndLon, tr.EndOdo, tr.DistanceKm, tr.Open = &end, fp(44.85), fp(-85.50), fp(1009), 9, false
	if err := srv.S.UpdateTrip(tr); err != nil {
		t.Fatal(err)
	}
	// the batch finds a road far longer than the odometer and even the straight line allow, and says so, so the owner can choose it
	code, body := do(t, admin, "POST", ts.URL+"/api/trips/estimate-routes", "", xh)
	if code != 200 || !strings.Contains(body, `"roadKm":30`) || !strings.Contains(body, `"odoKm":9`) {
		t.Fatalf("the failure carries both lengths: %d %s", code, body)
	}
	url := fmt.Sprintf("%s/api/trips/%d/distance", ts.URL, tr.ID)
	if code, _ := do(t, admin, "POST", url, `{"use":"road"}`, xh); code != 200 {
		t.Fatalf("use the road: %d", code)
	}
	got, _ := srv.S.GetTrip(tr.ID)
	if got.DistanceKm != 30 || got.DistanceSrc != "road" {
		t.Fatalf("corrected to the road: %+v", got)
	}
	if pts, _ := srv.S.RouteEstimate(tr.ID); len(pts) < 2 {
		t.Fatal("the road is kept so it can be drawn")
	}
	// the drive straight from start to end, offered beside the road through the recorded positions
	if code, body := do(t, admin, "GET", fmt.Sprintf("%s/api/trips/%d/road", ts.URL, tr.ID), "", xh); code != 200 || !strings.Contains(body, `"directKm":30`) || !strings.Contains(body, `"straightKm":`) {
		t.Fatalf("road and direct: %d %s", code, body)
	}
	if code, _ := do(t, admin, "POST", url, `{"use":"direct"}`, xh); code != 200 {
		t.Fatal("use the direct drive")
	}
	if got, _ = srv.S.GetTrip(tr.ID); got.DistanceSrc != "direct" {
		t.Fatalf("direct: %+v", got)
	}
	if code, _ := do(t, admin, "POST", url, `{"km":16.5}`, xh); code != 200 {
		t.Fatal("type a distance")
	}
	if got, _ = srv.S.GetTrip(tr.ID); got.DistanceKm != 16.5 || got.DistanceSrc != "manual" {
		t.Fatalf("typed: %+v", got)
	}
	if code, _ := do(t, admin, "POST", url, `{"use":"odometer"}`, xh); code != 200 {
		t.Fatal("back to the odometer")
	}
	if got, _ = srv.S.GetTrip(tr.ID); got.DistanceKm != 9 || got.DistanceSrc != "" {
		t.Fatalf("back to what the odometer said: %+v", got)
	}
	for _, bad := range []string{`{"km":0}`, `{"km":5000}`, `{}`} {
		if code, _ := do(t, admin, "POST", url, bad, xh); code != 400 {
			t.Fatalf("%s should be refused: %d", bad, code)
		}
	}
	// a viewer cannot change a distance
	if code, _ := do(t, &http.Client{}, "POST", url, `{"km":1}`, xh); code == 200 {
		t.Fatal("signed-out callers cannot")
	}
}
