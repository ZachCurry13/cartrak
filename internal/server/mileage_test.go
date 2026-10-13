package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/mileage"
	"github.com/ZachCurry13/cartrak/internal/settings"
	"github.com/ZachCurry13/cartrak/internal/store"
)

// nominatim is a fake lookup service: it names the place by the rounded coordinates it is sent.
func nominatim(t *testing.T) (*httptest.Server, *int32) {
	var hits int32
	cities := map[string]string{"12.100,23.100": "Springfield", "12.400,23.400": "Oakdale", "12.200,23.200": "Rivertown"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		key := r.URL.Query().Get("lat") + "," + r.URL.Query().Get("lon")
		w.Write([]byte(`{"address":{"city":"` + cities[key] + `"}}`))
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

func addTrip(t *testing.T, srv *Server, start time.Time, km float64, from, to [2]float64, cat string) {
	end := start.Add(20 * time.Minute).Unix()
	tr := &store.Trip{StartTS: start.Unix(), StartLat: &from[0], StartLon: &from[1], Category: cat, CategorySource: "default", TZ: "UTC"}
	if err := srv.S.CreateTrip(tr); err != nil {
		t.Fatal(err)
	}
	tr.EndTS, tr.EndLat, tr.EndLon, tr.DistanceKm, tr.Open = &end, &to[0], &to[1], km, false
	if err := srv.S.UpdateTrip(tr); err != nil {
		t.Fatal(err)
	}
}

func TestMileageReportForCalendarWeeks(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	srv.GeoGap = time.Millisecond
	svc, hits := nominatim(t)
	srv.Cfg.Set(settings.GeocodeURLKey, svc.URL)
	admin := adminClient(t, ts.URL)

	home, plym, rog := [2]float64{12.10, 23.10}, [2]float64{12.40, 23.40}, [2]float64{12.20, 23.20}
	// Mon Jun 22 2026 is the first day of ISO week 26.
	day := time.Date(2026, 6, 22, 14, 0, 0, 0, time.UTC)
	addTrip(t, srv, day, 40, home, rog, "work")
	addTrip(t, srv, day.Add(2*time.Hour), 40, rog, plym, "work")
	addTrip(t, srv, day.Add(4*time.Hour), 50, plym, home, "work")
	addTrip(t, srv, day.Add(6*time.Hour), 10, home, plym, "personal")                      // not a work trip
	addTrip(t, srv, time.Date(2026, 7, 14, 14, 0, 0, 0, time.UTC), 99, home, plym, "work") // week 29: out of range

	code, body := do(t, admin, "GET", ts.URL+"/api/mileage?weeks=26-28&year=2026", "", nil)
	if code != 200 {
		t.Fatalf("report = %d %s", code, body)
	}
	var out struct {
		From, To string
		Report   mileage.Report
		TSV      string `json:"tsv"`
		TSVFull  string `json:"tsvFull"`
	}
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		t.Fatal(err)
	}
	if out.From != "2026-06-22" || out.To != "2026-07-12" {
		t.Fatalf("weeks 26-28 are Jun 22 to Jul 12: %s to %s", out.From, out.To)
	}
	if len(out.Report.Rows) != 1 {
		t.Fatalf("rows: %+v", out.Report.Rows)
	}
	r := out.Report.Rows[0]
	if r.Date != "2026-06-22" || r.Origin != "Springfield" || r.Destinations != "Rivertown, Oakdale" || r.Miles != 81 || r.Description != "Customer Visits" {
		t.Fatalf("the row: %+v", r)
	}
	if out.TSV != "2026-06-22\tSpringfield\tRivertown, Oakdale\tCustomer Visits\t81\n" || !strings.HasSuffix(out.TSVFull, "Total Mileage Recorded:\t\t\t\t81\n") {
		t.Fatalf("copy blocks: %q / %q", out.TSV, out.TSVFull)
	}
	if *hits != 3 {
		t.Fatalf("three distinct places should be looked up once each, got %d", *hits)
	}

	// A second run answers from the remembered places and asks nobody.
	do(t, admin, "GET", ts.URL+"/api/mileage?from=2026-06-22&to=2026-06-28&description=Sales%20Calls", "", nil)
	_, again := do(t, admin, "GET", ts.URL+"/api/mileage?from=2026-06-22&to=2026-06-28&description=Sales%20Calls", "", nil)
	if *hits != 3 || !strings.Contains(again, "Sales Calls") {
		t.Fatalf("remembered places must not be looked up again (%d hits): %s", *hits, again)
	}
	if _, all := do(t, admin, "GET", ts.URL+"/api/mileage?weeks=26&year=2026&category=all", "", nil); !strings.Contains(all, `"trips":4`) {
		t.Fatalf("category=all includes personal trips: %s", all)
	}
}

func TestMileageReportInputsAndAccess(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	for _, bad := range []string{"", "?weeks=30-20", "?weeks=x", "?weeks=1&year=abc", "?from=2026-09-15", "?from=2026-09-15&to=2026-09-01", "?from=nope&to=nope", "?weeks=53&year=2025"} {
		if code, _ := do(t, admin, "GET", ts.URL+"/api/mileage"+bad, "", nil); code != 400 {
			t.Errorf("%q should be a 400, got %d", bad, code)
		}
	}
	do(t, admin, "POST", ts.URL+"/api/users", `{"name":"fam","password":"viewer-password-1","role":"viewer"}`, xh)
	vj, _ := cookieJar()
	v := &http.Client{Jar: vj}
	do(t, v, "POST", ts.URL+"/api/login", `{"name":"fam","password":"viewer-password-1"}`, nil)
	if code, _ := do(t, v, "GET", ts.URL+"/api/mileage?weeks=1", "", nil); code != 403 {
		t.Fatalf("a viewer built a report (it can send locations to a lookup service): %d", code)
	}
	_ = srv
}

func TestMileageNeverAsksWhenLookupIsSwitchedOff(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	svc, hits := nominatim(t)
	srv.Cfg.Set(settings.GeocodeURLKey, svc.URL)
	srv.Cfg.Set(settings.GeocodeEnabledKey, "off")
	admin := adminClient(t, ts.URL)
	home, plym := [2]float64{12.10, 23.10}, [2]float64{12.40, 23.40}
	addTrip(t, srv, time.Date(2026, 6, 22, 14, 0, 0, 0, time.UTC), 40, home, plym, "work")
	srv.S.SetPlace("12.400,23.400", "Oakdale") // one place is already remembered

	_, body := do(t, admin, "GET", ts.URL+"/api/mileage?weeks=26&year=2026", "", nil)
	if *hits != 0 {
		t.Fatalf("with lookup off nothing may be sent anywhere: %d requests", *hits)
	}
	if !strings.Contains(body, `"destinations":"Oakdale"`) || !strings.Contains(body, "switched off") || !strings.Contains(body, `"lookupOn":false`) {
		t.Fatalf("remembered places still work, and the page is told why the rest are missing: %s", body)
	}
	if settings.Valid(settings.GeocodeURLKey, "http://evil.example.com") == nil || settings.Valid(settings.GeocodeEnabledKey, "maybe") == nil {
		t.Fatal("lookup settings must be validated")
	}
}
