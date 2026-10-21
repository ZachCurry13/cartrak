package server

import (
	"github.com/ZachCurry13/cartrak/internal/geocode"
	"github.com/ZachCurry13/cartrak/internal/store"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMessagesFollowTheUnitsTheDeviceSends(t *testing.T) {
	imperial := httptest.NewRequest("GET", "/", nil)
	metric := httptest.NewRequest("GET", "/", nil)
	metric.Header.Set("X-Units", "metric")
	if metricReq(imperial) || !metricReq(metric) {
		t.Fatal("miles are the default, kilometers only when asked")
	}
	f := routeFailure{RoadKm: 18, OdoKm: 9}
	if got := f.text(false); got != "the road found was 11.2 mi but the odometer says 5.6 mi" {
		t.Fatal(got)
	}
	if got := f.text(true); got != "the road found was 18.0 km but the odometer says 9.0 km" {
		t.Fatal(got)
	}
	if speedText(100, false) != "62 mph" || distText(160.9344, false, 0) != "100 mi" {
		t.Fatal("conversion")
	}
	in := maintIn{EveryKm: new(float64)}
	*in.EveryKm = 5
	if msg := in.valid(false); !strings.Contains(msg, "miles") || strings.Contains(msg, " km") {
		t.Fatal(msg)
	}
}

func TestChargingSettingsHaveTheirOwnListAndPlacesCarryTheirOwnPrice(t *testing.T) {
	ts, cfg := newCfgServer(t, nil)
	c := adminClient(t, ts.URL)
	_, general := do(t, c, "GET", ts.URL+"/api/admin/settings", "", nil)
	_, charging := do(t, c, "GET", ts.URL+"/api/admin/settings?group=charging", "", nil)
	if strings.Contains(general, "CARTRAK_PRICE_HOME") || !strings.Contains(general, "FORD_CLIENT_ID") {
		t.Fatalf("the general list must not carry the charging settings: %s", general)
	}
	if !strings.Contains(charging, "CARTRAK_PRICE_HOME") || strings.Contains(charging, "FORD_CLIENT_ID") {
		t.Fatalf("the charging list holds only charging settings: %s", charging)
	}
	if code, _ := do(t, c, "PUT", ts.URL+"/api/admin/settings?group=charging", `{"values":{"CARTRAK_PRICE_HOME":"0.12"}}`, xh); code != 200 || cfg.PriceHome() == nil || *cfg.PriceHome() != 0.12 {
		t.Fatal("saving a charging setting")
	}
	// a named place with its own price
	if code, body := do(t, c, "POST", ts.URL+"/api/named-places", `{"name":"Cabin","lat":45.1,"lon":-90.2,"chargePrice":0.2}`, xh); code != 200 {
		t.Fatalf("add place: %d %s", code, body)
	}
	_, list := do(t, c, "GET", ts.URL+"/api/named-places", "", nil)
	if !strings.Contains(list, `"chargePrice":0.2`) {
		t.Fatalf("the place keeps its price: %s", list)
	}
	if code, _ := do(t, c, "POST", ts.URL+"/api/named-places", `{"name":"Bad","lat":45.1,"lon":-90.2,"chargePrice":99}`, xh); code != 400 {
		t.Fatal("an absurd price is refused")
	}
}

func TestAReportCanBeDownloadedForSomeCategoriesAndADateRange(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	end1, end2, end3 := int64(1773500000+1800), int64(1773600000+1800), int64(1773700000+1800)
	for _, x := range []struct {
		start, end int64
		cat        string
		km         float64
	}{{1773500000, end1, "work", 16.0934}, {1773600000, end2, "medical", 8.0467}, {1773700000, end3, "personal", 32.1868}} {
		tr := &store.Trip{StartTS: x.start, Category: x.cat, CategorySource: "manual"}
		_ = srv.S.CreateTrip(tr)
		tr.EndTS, tr.DistanceKm, tr.Open = &x.end, x.km, false
		_ = srv.S.UpdateTrip(tr)
	}
	day := time.Unix(1773500000, 0).In(srv.defaultLoc()).Format("2006-01-02")
	code, body := do(t, admin, "GET", ts.URL+"/api/reports/trips.csv?categories=work,medical&from="+day+"&to=2027-01-01&columns=purpose,distance", "", nil)
	if code != 200 || !strings.Contains(body, "Work,10") || !strings.Contains(body, "Medical,5") || strings.Contains(body, "Personal") {
		t.Fatalf("two categories chosen: %d %q", code, body)
	}
	code, body = do(t, admin, "GET", ts.URL+"/api/reports/trips.xlsx?from="+day+"&to=2027-01-01", "", nil)
	if code != 200 || !strings.HasPrefix(body, "PK") {
		t.Fatalf("an Excel file is a zip: %d", code)
	}
	if code, _ := do(t, admin, "GET", ts.URL+"/api/reports/trips.csv", "", nil); code != 400 {
		t.Fatalf("a range is required: %d", code)
	}
	// a custom report keeps its layout
	save := `{"name":"Customer report","config":{"columns":[{"key":"date","label":"Day"},{"key":"distance","label":"Mi"}],"fixed":[{"label":"Client","value":"Acme"}],"summary":false}}`
	if code, b := do(t, admin, "POST", ts.URL+"/api/reports/templates", save, xh); code != 200 {
		t.Fatalf("save template: %d %s", code, b)
	}
	_, opts := do(t, admin, "GET", ts.URL+"/api/reports/options", "", nil)
	if !strings.Contains(opts, "Customer report") || !strings.Contains(opts, `"key":"value"`) {
		t.Fatalf("options: %s", opts)
	}
	_, body = do(t, admin, "GET", ts.URL+"/api/reports/trips.csv?template=1&categories=work&from="+day+"&to=2027-01-01", "", nil)
	if !strings.HasPrefix(body, "Day,Mi,Client\n") || !strings.Contains(body, ",10,Acme") {
		t.Fatalf("the saved layout is used: %q", body)
	}
	if code, _ := do(t, admin, "POST", ts.URL+"/api/reports/templates", `{"name":"x","config":{"columns":[{"key":"bogus"}]}}`, xh); code != 400 {
		t.Fatal("an unknown column is refused")
	}
}

func TestTheFedExReportIsTheMileageLogAndStandardReportsNameJustTheCity(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	f := func(v float64) *float64 { return &v }
	home, shop := [2]float64{45.16, -93.86}, [2]float64{45.02, -93.45}
	_ = srv.S.SetPlace(geocode.Key(home[0], home[1]), "Buffalo")
	_ = srv.S.SetPlace(geocode.Key(shop[0], shop[1]), "Plymouth")
	end := int64(1773500000 + 1800)
	tr := &store.Trip{StartTS: 1773500000, Category: "work", CategorySource: "manual", StartLat: f(home[0]), StartLon: f(home[1])}
	_ = srv.S.CreateTrip(tr)
	tr.EndTS, tr.DistanceKm, tr.Open, tr.EndLat, tr.EndLon = &end, 16.0934, false, f(shop[0]), f(shop[1])
	_ = srv.S.UpdateTrip(tr)
	day := time.Unix(1773500000, 0).In(srv.defaultLoc()).Format("2006-01-02")
	_, body := do(t, admin, "GET", ts.URL+"/api/reports/trips.csv?template=fedex&from="+day+"&to="+day, "", nil)
	lines := strings.Split(strings.TrimSpace(body), "\n")
	if lines[0] != "Date,Origin City,Destination City,Description,Total Miles Driven" || lines[1] != day+",Buffalo,Plymouth,Customer Visits,10" || !strings.Contains(body, "Total Mileage Recorded:,,,,10") {
		t.Fatalf("the mileage log layout: %q", body)
	}
	_, body = do(t, admin, "GET", ts.URL+"/api/reports/trips.csv?from="+day+"&to="+day+"&columns=from,to,distance", "", nil)
	if !strings.Contains(body, "Buffalo,Plymouth,10") {
		t.Fatalf("just the city: %q", body)
	}
	_, body = do(t, admin, "POST", ts.URL+"/api/reports/lookup?from="+day+"&to="+day, "", xh)
	if !strings.Contains(body, `"pending":0`) {
		t.Fatalf("everything is already named: %s", body)
	}
}
