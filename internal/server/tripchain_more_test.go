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

func chainServers(t *testing.T) (*httptest.Server, *Server, *http.Client) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	seedUsualDriving(t, srv)
	tesla := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var s []string
		for _, lat := range []float64{45.8, 47.3} {
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

func TestTripChainOffersElectrifyAmericaAsAnOptionAndUsesItWhenAsked(t *testing.T) {
	ts, srv, admin := chainServers(t)
	post := func(body string) (int, string) { return do(t, admin, "POST", ts.URL+"/api/trip-chain", body, xh) }
	if c, _ := post(chainBody(`,"ea":true`)); c != 400 {
		t.Fatalf("Electrify America without a key gave %d", c)
	}
	srv.Cfg.Set(settings.AFDCKeyKey, "test-key")
	_, out := post(chainBody(``))
	var r chainOut
	json.Unmarshal([]byte(out), &r)
	if r.Alt == nil || r.Alt.MinutesSaved < 10 || strings.Contains(out, "Electrify America") {
		t.Fatalf("Tesla only should name no other network but say what it would save: %s", out)
	}
	_, out = post(chainBody(`,"ea":true`))
	if !strings.Contains(out, "Electrify America: Mall") || strings.Contains(out, `"alt"`) {
		t.Fatalf("with Electrify America: %s", out)
	}
	if c, _ := post(chainBody(`,"avoid":["45.8000,-93.2000"]`)); c != 422 {
		t.Fatalf("avoiding a needed Supercharger gave %d", c)
	}
}

func TestTripChainPassesThePlacesGiven(t *testing.T) {
	ts, _, admin := chainServers(t)
	_, out := do(t, admin, "POST", ts.URL+"/api/trip-chain", chainBody(`,"via":[{"name":"Escanaba","lat":46.5,"lon":-93.5,"charge":false}]`), xh)
	if !strings.Contains(out, `"name":"Escanaba"`) {
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
