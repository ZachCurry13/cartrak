package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ZachCurry13/cartrak/internal/settings"
)

func chainBody(extra string) string {
	return `{"start":{"name":"Home","lat":45,"lon":-93.2},"dest":{"name":"There","lat":48.5,"lon":-93.2},"startPct":90,"reservePct":10,"targetPct":90,"chargeKw":150,"maxLegMin":180,"dayMin":720,"plugs":"nacs"` + extra + `}`
}

type chainOut struct {
	Days []struct {
		Stops []struct {
			Name, Kind string
			Lat        float64
		}
	}
	Alt *struct {
		MinutesSaved float64
		ChargeStops  int
	}
}

func chainServers(t *testing.T, teslaLats ...float64) (*httptest.Server, *Server, *http.Client) {
	if len(teslaLats) == 0 {
		teslaLats = []float64{45.8, 47.3}
	}
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	seedUsualDriving(t, srv)
	tesla := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var s []string
		for _, lat := range teslaLats {
			s = append(s, fmt.Sprintf(`{"name":"Super %.1f","status":"OPEN","gps":{"latitude":%.1f,"longitude":-93.2},"powerKilowatt":250,"otherEVs":true,"plugshareId":4242,"plugs":{"nacs":8}}`, lat, lat))
		}
		w.Write([]byte("[" + strings.Join(s, ",") + "]"))
	}))
	t.Cleanup(tesla.Close)
	ea := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"fuel_stations":[{"station_name":"Mall","latitude":46.75,"longitude":-93.2,"ev_dc_fast_num":6,"ev_connector_types":["J1772COMBO"]}]}`))
	}))
	t.Cleanup(ea.Close)
	srv.SuperURL, srv.AFDCURL = tesla.URL, ea.URL
	return ts, srv, admin
}

func TestTripChainUsesElectrifyAmericaOnlyWhenAskedAndSaysWhatItWouldSave(t *testing.T) {
	// Superchargers at 45.8 and 47.5 need two stops; the Electrify America charger at 46.75 does it in one
	ts, srv, admin := chainServers(t, 45.9, 47.5)
	post := func(body string) (int, string) { return do(t, admin, "POST", ts.URL+"/api/trip-chain", body, xh) }
	if c, _ := post(chainBody(`,"ea":true`)); c != 400 {
		t.Fatalf("Electrify America without a key gave %d", c)
	}
	srv.Cfg.Set(settings.AFDCKeyKey, "test-key")
	_, out := post(chainBody(``))
	var r chainOut
	json.Unmarshal([]byte(out), &r)
	if r.Alt == nil || r.Alt.MinutesSaved < 5 || strings.Contains(out, "Electrify America") {
		t.Fatalf("Tesla only should name no other network but say what it would save: %s", out)
	}
	if _, out = post(chainBody(`,"ea":true`)); !strings.Contains(out, `"days"`) || strings.Contains(out, `"alt"`) {
		t.Fatalf("with Electrify America: %s", out)
	}
	if c, _ := post(chainBody(`,"avoid":["45.9000,-93.2000"]`)); c != 422 {
		t.Fatalf("avoiding a needed Supercharger gave %d", c)
	}
}

func TestTripChainReachesAGapOnlyElectrifyAmericaFills(t *testing.T) {
	ts, srv, admin := chainServers(t, 45.8) // the next Supercharger is too far away
	srv.Cfg.Set(settings.AFDCKeyKey, "test-key")
	if c, _ := do(t, admin, "POST", ts.URL+"/api/trip-chain", chainBody(``), xh); c != 422 {
		t.Fatalf("Tesla only has a gap: %d", c)
	}
	_, out := do(t, admin, "POST", ts.URL+"/api/trip-chain", chainBody(`,"ea":true`), xh)
	if !strings.Contains(out, "Electrify America: Mall") {
		t.Fatalf("the Electrify America charger fills the gap: %s", out)
	}
}

func TestTripChainPassesThePlacesGiven(t *testing.T) {
	ts, _, admin := chainServers(t)
	_, out := do(t, admin, "POST", ts.URL+"/api/trip-chain", chainBody(`,"via":[{"name":"Midway","lat":46.5,"lon":-93.5,"charge":false}]`), xh)
	if !strings.Contains(out, `"name":"Midway"`) {
		t.Fatalf("the place to pass is missing: %s", out)
	}
	if c, _ := do(t, admin, "POST", ts.URL+"/api/trip-chain", chainBody(`,"via":[{"name":"Nowhere","lat":0,"lon":0}]`), xh); c != 400 {
		t.Fatalf("an unknown place gave %d", c)
	}
}

func TestAHotelNightNamesTheNearestCharger(t *testing.T) {
	ts, srv, admin := chainServers(t)
	srv.Cfg.Set(settings.AFDCKeyKey, "test-key")
	_, out := do(t, admin, "POST", ts.URL+"/api/trip-chain", chainBody(`,"ea":true,"via":[{"name":"Hotel","lat":46.7,"lon":-93.25,"charge":false,"night":true}]`), xh)
	var r struct {
		Days []struct {
			Stops []struct {
				Name string
				Kind string
				Near *struct {
					Name string
					Km   float64
				}
			}
		}
	}
	if json.Unmarshal([]byte(out), &r) != nil || len(r.Days) != 2 {
		t.Fatalf("want two days around the hotel: %s", out)
	}
	night := r.Days[0].Stops[len(r.Days[0].Stops)-1]
	if night.Name != "Hotel" || night.Near == nil || !strings.Contains(night.Near.Name, "Electrify America: Mall") || night.Near.Km > 10 {
		t.Fatalf("the hotel should name the charger close by: %s", out)
	}
}

func TestAPlaceCanBeTurnedIntoTheClosestCharger(t *testing.T) {
	ts, _, admin := chainServers(t)
	post := func(via string) (int, string) {
		return do(t, admin, "POST", ts.URL+"/api/trip-chain", chainBody(`,"via":[`+via+`]`), xh)
	}
	c, out := post(`{"name":"Town","lat":47.2,"lon":-93.25,"mode":"charge"}`)
	if c != 200 || !strings.Contains(out, `"name":"Supercharger: Super 47.3"`) || strings.Contains(out, `"name":"Town"`) {
		t.Fatalf("the place should become the closest charger: %d %s", c, out)
	}
	if c, out = post(`{"name":"Faraway","lat":40,"lon":-80,"mode":"charge"}`); c != 422 || !strings.Contains(out, "within") {
		t.Fatalf("no charger near the place: %d %s", c, out)
	}
	if c, out = post(`{"name":"Roadside","lat":47.0,"lon":-93.2,"mode":"pass"}`); c != 200 || !strings.Contains(out, `"name":"Roadside"`) {
		t.Fatalf("a place to drive through stays as it is: %d %s", c, out)
	}
	_, src := do(t, admin, "GET", ts.URL+"/api/trip-chain/sources", "", nil)
	if !strings.Contains(src, `"ready":true`) || !strings.Contains(src, `"adapter":2`) {
		t.Fatalf("the counts: %s", src)
	}
}

func TestTheRangeBetweenTwoChargeLevelsComesFromTheCarsOwnDriving(t *testing.T) {
	ts, _ := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	if _, b := do(t, admin, "GET", ts.URL+"/api/trip-chain/range", "", nil); !strings.Contains(b, `"ready":false`) {
		t.Fatalf("with no driving yet there is nothing to say: %s", b)
	}
}

func TestTheRangeUsesTheChargeTargetAndReserveAndOffersOtherIdeas(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	seedUsualDriving(t, srv) // 90 kWh at 0.17 kWh/km
	_, b := do(t, admin, "GET", ts.URL+"/api/trip-chain/range?reserve=20&target=80", "", nil)
	var r struct {
		Ready   bool
		Windows []struct{ From, To, Km, Minutes float64 }
	}
	if json.Unmarshal([]byte(b), &r) != nil || !r.Ready || len(r.Windows) < 3 {
		t.Fatalf("range: %s", b)
	}
	first := r.Windows[0] // 80 to 20 percent of 90 kWh is 54 kWh, about 318 km at 0.17 kWh/km
	if first.From != 80 || first.To != 20 || first.Km < 250 || first.Km > 380 {
		t.Fatalf("the first window is the planner's own settings: %+v", first)
	}
	seen := map[[2]float64]bool{}
	for _, w := range r.Windows {
		if seen[[2]float64{w.From, w.To}] {
			t.Fatalf("a window appears twice: %+v", r.Windows)
		}
		seen[[2]float64{w.From, w.To}] = true
	}
}
