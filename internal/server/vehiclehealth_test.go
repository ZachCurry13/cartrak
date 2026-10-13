package server

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/vhealth"
)

func TestVehicleHealthEndpointForABatteryCar(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)

	_, b := do(t, admin, "GET", ts.URL+"/api/vehicle-health", "", nil)
	if !strings.Contains(b, `"ready":false`) {
		t.Fatalf("before the first reading: %s", b)
	}

	raw, err := os.ReadFile("../ford/testdata/bev_response.json")
	if err != nil {
		t.Fatal(err)
	}
	st := ford.ParseState(raw, time.Now())
	srv.S.SaveState("test", st)
	srv.P.LearnCaps(st) // learns BEV from the garage record

	_, b = do(t, admin, "GET", ts.URL+"/api/vehicle-health", "", nil)
	var out struct {
		Vehicle struct{ Powertrain, Source string }
		Model   string
		Report  vhealth.Report
	}
	if err := json.Unmarshal([]byte(b), &out); err != nil {
		t.Fatal(err)
	}
	if out.Vehicle.Powertrain != "ev" || out.Vehicle.Source != "garage" || out.Model != "2021 Mustang Mach-E" {
		t.Fatalf("vehicle: %+v model %q", out.Vehicle, out.Model)
	}
	ids := map[string]bool{}
	for _, s := range out.Report.Sections {
		ids[s.ID] = true
	}
	if !ids["12v"] || !ids["traction"] || !ids["warnings"] || ids["engine"] {
		t.Fatalf("sections for a battery car: %v", ids)
	}
	if strings.Contains(b, "TESTVIN") || strings.Contains(strings.ToLower(b), `"lat"`) {
		t.Fatalf("the health report must never carry the VIN or location: %s", b)
	}
	if code, _ := do(t, &http.Client{}, "GET", ts.URL+"/api/vehicle-health", "", nil); code != 401 {
		t.Fatalf("anonymous: %d", code)
	}
}
