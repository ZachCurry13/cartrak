package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/settings"
	"github.com/ZachCurry13/cartrak/internal/store"
)

func TestCompanyListFindsRemembersAndAsks(t *testing.T) {
	svc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("q") == "Acme Widgets" {
			w.Write([]byte(`[{"lat":"44.9","lon":"-93.2","display_name":"Acme Widgets, 1 Main St, Townsville","name":"Acme Widgets","type":"office"}]`))
			return
		}
		w.Write([]byte(`[{"lat":"45.1","lon":"-93.3","display_name":"Something Else, 9 Oak St, Townsville","name":"Something Else"}]`))
	}))
	defer svc.Close()
	ts, srv := healthServer(t, fakeProv{}, false, false)
	srv.GeoGap = time.Millisecond
	srv.Cfg.Set(settings.GeocodeURLKey, svc.URL)
	admin := adminClient(t, ts.URL)
	type resp struct {
		Rows []struct{ Name, Ref, Status string }
		More int
	}
	ask := func(text string) resp {
		b, _ := json.Marshal(map[string]string{"text": text})
		code, body := do(t, admin, "POST", ts.URL+"/api/company-list/resolve", string(b), xh)
		var r resp
		if code != 200 || json.Unmarshal([]byte(body), &r) != nil {
			t.Fatalf("resolve: %d %s", code, body)
		}
		return r
	}
	r := ask("Acme Widgets-1234567-U.S.A.\nMystery Metals-7654321-U.S.A.")
	if len(r.Rows) != 2 || r.Rows[0].Status != "found" || r.Rows[1].Status != "check" || r.Rows[1].Ref != "7654321" {
		t.Fatalf("first: %+v", r)
	}
	if c, _ := do(t, admin, "POST", ts.URL+"/api/company-list/remember", `{"ref":"7654321","name":"Mystery Metals","address":"9 Oak St","lat":45.1,"lon":-93.3}`, xh); c != 200 {
		t.Fatalf("remember: %d", c)
	}
	if c, _ := do(t, admin, "POST", ts.URL+"/api/company-list/remember", `{"ref":"x1","name":"Bad","lat":1,"lon":1}`, xh); c != 400 {
		t.Fatalf("a non-number ref should be refused, got %d", c)
	}
	if r = ask("Mystery Metals-7654321-U.S.A."); r.Rows[0].Status != "remembered" {
		t.Fatalf("second: %+v", r)
	}
}

func TestAPlaceOnAnotherContinentIsNeverOfferedAndANearMatchIsAskedForCountry(t *testing.T) {
	var gotCountry string
	svc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCountry = r.URL.Query().Get("countrycodes")
		switch r.URL.Query().Get("q") {
		case "Faraway Goods": // the same name, far from Home
			w.Write([]byte(`[{"lat":"49.99","lon":"36.23","display_name":"Faraway Goods, Kharkiv","name":"Faraway Goods","type":"office"}]`))
		case "Nearby Goods":
			w.Write([]byte(`[{"lat":"44.95","lon":"-93.25","display_name":"Nearby Goods, 1 Main St, Townsville","name":"Nearby Goods","type":"office"}]`))
		default:
			w.Write([]byte(`[]`))
		}
	}))
	defer svc.Close()
	ts, srv := healthServer(t, fakeProv{}, false, false)
	srv.GeoGap = time.Millisecond
	srv.Cfg.Set(settings.GeocodeURLKey, svc.URL)
	if err := srv.S.SaveNamedPlace(&store.NamedPlace{Name: "Home", Lat: 44.9, Lon: -93.2, RadiusM: 150}); err != nil {
		t.Fatal(err)
	}
	admin := adminClient(t, ts.URL)
	b, _ := json.Marshal(map[string]string{"text": "Faraway Goods, Inc.-111-U.S.A.\nNearby Goods LLC-222-U.S.A."})
	code, body := do(t, admin, "POST", ts.URL+"/api/company-list/resolve", string(b), xh)
	var r struct {
		Rows []struct {
			Name, Status string
			Options      []struct{ Name string }
		}
	}
	if code != 200 || json.Unmarshal([]byte(body), &r) != nil || len(r.Rows) != 2 {
		t.Fatalf("resolve: %d %s", code, body)
	}
	if r.Rows[0].Status != "none" || len(r.Rows[0].Options) != 0 {
		t.Fatalf("a place on another continent is not offered: %+v", r.Rows[0])
	}
	if r.Rows[1].Status != "found" {
		t.Fatalf("a near place with the same name is taken: %+v", r.Rows[1])
	}
	if gotCountry != "us" {
		t.Fatalf("US units keep the search to the United States: %q", gotCountry)
	}
}

func TestWithAGooglePlacesKeyACompanyIsFoundThereFirstAndKeyIsNeverSent(t *testing.T) {
	var gotKey string
	google := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("X-Goog-Api-Key")
		w.Write([]byte(`{"places":[{"displayName":{"text":"Acme Widgets"},"formattedAddress":"1 Main St, Townsville, MN","location":{"latitude":44.95,"longitude":-93.25}}]}`))
	}))
	defer google.Close()
	ts, srv := healthServer(t, fakeProv{}, false, false)
	srv.PlacesURL = google.URL
	srv.Cfg.Set(settings.GooglePlacesKeyKey, "secret-key")
	if err := srv.S.SaveNamedPlace(&store.NamedPlace{Name: "Home", Lat: 44.9, Lon: -93.2, RadiusM: 150}); err != nil {
		t.Fatal(err)
	}
	admin := adminClient(t, ts.URL)
	b, _ := json.Marshal(map[string]string{"text": "Acme Widgets, Inc.-111-U.S.A."})
	code, body := do(t, admin, "POST", ts.URL+"/api/company-list/resolve", string(b), xh)
	if code != 200 || gotKey != "secret-key" || !strings.Contains(body, `"status":"found"`) {
		t.Fatalf("%d %q %s", code, gotKey, body)
	}
	if strings.Contains(body, "secret-key") {
		t.Fatal("the key must never be sent to the browser")
	}
}
