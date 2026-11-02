package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/settings"
)

func TestAddressSearchFindsAPlaceAndSavesItByCoordinates(t *testing.T) {
	var asked string
	svc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked = r.URL.Query().Get("q")
		w.Write([]byte(`[{"lat":"44.9778","lon":"-93.2650","display_name":"350 5th Ave, Townsville"}]`))
	}))
	defer svc.Close()
	ts, srv := healthServer(t, fakeProv{}, false, false)
	srv.GeoGap = time.Millisecond
	srv.Cfg.Set(settings.GeocodeURLKey, svc.URL)
	admin := adminClient(t, ts.URL)

	code, body := do(t, admin, "GET", ts.URL+"/api/named-places/search?q="+url.QueryEscape("350 5th Ave Townsville"), "", nil)
	var found []struct {
		Label    string
		Lat, Lon float64
	}
	if code != 200 || json.Unmarshal([]byte(body), &found) != nil || len(found) != 1 || found[0].Lat != 44.9778 || asked != "350 5th Ave Townsville" {
		t.Fatalf("search: %d %s (asked %q)", code, body, asked)
	}
	if c, _ := do(t, admin, "POST", ts.URL+"/api/named-places", `{"name":"Customer","lat":44.9778,"lon":-93.265,"radiusM":150}`, xh); c != 200 {
		t.Fatalf("saving the found place: %d", c)
	}
	_, list := do(t, admin, "GET", ts.URL+"/api/named-places", "", nil)
	var ps []struct{ Name string }
	json.Unmarshal([]byte(list), &ps)
	if len(ps) != 1 || ps[0].Name != "Customer" {
		t.Fatalf("places: %s", list)
	}
}

func TestAddressSearchStopsWhenCityNamesAreOff(t *testing.T) {
	hit := false
	svc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hit = true }))
	defer svc.Close()
	ts, srv := healthServer(t, fakeProv{}, false, false)
	srv.Cfg.Set(settings.GeocodeURLKey, svc.URL)
	srv.Cfg.Set(settings.GeocodeEnabledKey, "off")
	admin := adminClient(t, ts.URL)
	if code, _ := do(t, admin, "GET", ts.URL+"/api/named-places/search?q=1+Main+St", "", nil); code != 400 || hit {
		t.Fatalf("the lookup ran while off: %d, service asked %v", code, hit)
	}
}
