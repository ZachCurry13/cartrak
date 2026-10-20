package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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
