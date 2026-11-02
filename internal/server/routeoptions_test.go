package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/routeest"
	"github.com/ZachCurry13/cartrak/internal/settings"
	"github.com/ZachCurry13/cartrak/internal/store"
)

// A road service that offers three roads between any two places: 10 km (the quickest), 8 km and 12 km. The shortest is 8 km too, so it is not offered as another choice.
func threeRoadService() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"code":"Ok","routes":[
			{"distance":10000,"geometry":{"coordinates":[[-85.60,44.76],[-85.50,44.85]]}},
			{"distance":8000,"geometry":{"coordinates":[[-85.60,44.76],[-85.55,44.80],[-85.50,44.85]]}},
			{"distance":12000,"geometry":{"coordinates":[[-85.60,44.76],[-85.52,44.78],[-85.50,44.85]]}}]}`))
	}))
}

func TestRouteOptionsOfferTheRoadClosestToTheOdometerAndAChosenOneIsKeptEvenAfterAGiveUp(t *testing.T) {
	osrm := threeRoadService()
	defer osrm.Close()
	ts, srv := healthServer(t, fakeProv{}, false, false)
	srv.RouteEst = &routeest.Client{BaseURL: osrm.URL}
	srv.Cfg.Set(settings.RouteKey, "on")
	admin := adminClient(t, ts.URL)
	fp := func(v float64) *float64 { return &v }
	t0 := time.Now().Add(-3 * time.Hour).Truncate(time.Second)
	for i, p := range [][2]float64{{44.76, -85.60}, {44.85, -85.50}} {
		st := &ford.State{TS: t0.Add(time.Duration(i*20) * time.Minute), Lat: fp(p[0]), Lon: fp(p[1]), OdometerKm: fp(1000 + float64(i)*8.2), Raw: []byte(`{}`)}
		if _, err := srv.S.SaveState("t", st); err != nil {
			t.Fatal(err)
		}
	}
	end := t0.Add(20 * time.Minute).Unix()
	tr := &store.Trip{StartTS: t0.Unix(), StartLat: fp(44.76), StartLon: fp(-85.60), Category: "personal", CategorySource: "default"}
	if err := srv.S.CreateTrip(tr); err != nil {
		t.Fatal(err)
	}
	tr.EndTS, tr.EndLat, tr.EndLon, tr.DistanceKm, tr.Open = &end, fp(44.85), fp(-85.50), 8.2, false // the odometer says 8.2 km
	if err := srv.S.UpdateTrip(tr); err != nil {
		t.Fatal(err)
	}
	_ = srv.S.SaveNoRoute(tr.ID) // it was tried before and found unusable

	code, b := do(t, admin, "GET", fmt.Sprintf("%s/api/trips/%d/route-options", ts.URL, tr.ID), "", nil)
	var opts struct {
		OdoKm      float64
		OdoTrusted bool
		Options    []struct {
			Kind string
			Km   float64
		}
	}
	if code != 200 || json.Unmarshal([]byte(b), &opts) != nil || len(opts.Options) != 2 || !opts.OdoTrusted {
		t.Fatalf("%d %s", code, b)
	}
	if opts.Options[0].Kind != "closest" || opts.Options[0].Km != 8 || opts.Options[1].Kind != "quickest" || opts.Options[1].Km != 10 {
		t.Fatalf("8 km is nearest the odometer's 8.2: %+v", opts.Options)
	}

	if c, _ := do(t, admin, "POST", fmt.Sprintf("%s/api/trips/%d/route-choice", ts.URL, tr.ID), `{"kind":"nonsense"}`, xh); c != 404 {
		t.Fatalf("an unknown road is refused: %d", c)
	}
	if c, body := do(t, admin, "POST", fmt.Sprintf("%s/api/trips/%d/route-choice", ts.URL, tr.ID), `{"kind":"quickest"}`, xh); c != 200 {
		t.Fatalf("%d %s", c, body)
	}
	if pts, km := srv.S.RouteEstimate(tr.ID); len(pts) < 2 || km != 10 || srv.S.RouteGaveUp(tr.ID) {
		t.Fatalf("the chosen road replaces the give-up: %d points, %v km", len(pts), km)
	}
}

func TestRouteOptionsNeedTheRoadLookupSwitchedOn(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	end := time.Now().Add(-time.Hour).Unix()
	tr := &store.Trip{StartTS: end - 600, Category: "personal", CategorySource: "default"}
	if err := srv.S.CreateTrip(tr); err != nil {
		t.Fatal(err)
	}
	tr.EndTS, tr.Open, tr.DistanceKm = &end, false, 5
	srv.S.UpdateTrip(tr)
	if c, _ := do(t, admin, "GET", fmt.Sprintf("%s/api/trips/%d/route-options", ts.URL, tr.ID), "", nil); c != 409 {
		t.Fatalf("nothing is asked while the lookup is off: %d", c)
	}
}
