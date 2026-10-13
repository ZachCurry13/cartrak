package server

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/settings"
)

func TestVehicleTypeIsDetectedThenOverridden(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	get := func() string { _, b := do(t, admin, "GET", ts.URL+"/api/vehicle", "", nil); return b }

	if b := get(); !strings.Contains(b, `"source":"unknown"`) || !strings.Contains(b, `"engine":true`) || !strings.Contains(b, `"battery":true`) {
		t.Fatalf("before any data everything stays visible: %s", b)
	}
	srv.S.SetKV(ford.CapsKV, "fuel")
	if b := get(); !strings.Contains(b, `"powertrain":"gas"`) || !strings.Contains(b, `"battery":false`) || !strings.Contains(b, `"source":"detected"`) {
		t.Fatalf("a car with only fuel data is gas: %s", b)
	}
	srv.S.SetKV(ford.CapsKV, "ev,fuel,plug")
	if b := get(); !strings.Contains(b, `"powertrain":"phev"`) || !strings.Contains(b, `"plug":true`) || !strings.Contains(b, `"engine":true`) {
		t.Fatalf("plug-in hybrid: %s", b)
	}
	srv.Cfg.Set(settings.PowertrainKey, "ev")
	if b := get(); !strings.Contains(b, `"powertrain":"ev"`) || !strings.Contains(b, `"source":"setting"`) || !strings.Contains(b, `"engine":false`) {
		t.Fatalf("the admin's choice must win: %s", b)
	}
	if code, _ := do(t, &http.Client{}, "GET", ts.URL+"/api/vehicle", "", nil); code != 401 {
		t.Fatalf("anonymous: %d", code)
	}
	if settings.Valid(settings.PowertrainKey, "diesel-truck") == nil || settings.Valid(settings.PowertrainKey, "") != nil || settings.Valid(settings.PowertrainKey, "hev") != nil {
		t.Fatal("only the listed vehicle types (or empty) are accepted")
	}
}

func TestPollerLearnsWhatTheCarReports(t *testing.T) {
	_, srv := healthServer(t, fakeProv{}, false, false)
	st := ford.ParseState([]byte(`{"m":{"fuelLevel":{"value":50},"oilLifeRemaining":{"value":80}}}`), time.Now())
	srv.P.LearnCaps(st)
	srv.P.LearnCaps(ford.ParseState([]byte(`{"m":{"speed":{"value":1}}}`), time.Now())) // a thin reading changes nothing
	if p, src := srv.powertrain(); p != ford.Gas || src != "detected" {
		t.Fatalf("got %q from %s", p, src)
	}
}

func TestFordsGarageRecordBeatsGuessingFromMetrics(t *testing.T) {
	// A battery car also reports oil life and hybrid mode, so the garage record is trusted first.
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	srv.S.SetKV(ford.CapsKV, "ev,fuel,plug") // would look like a plug-in hybrid
	srv.S.SetKV(ford.EngineKV, "BEV")
	_, b := do(t, admin, "GET", ts.URL+"/api/vehicle", "", nil)
	if !strings.Contains(b, `"powertrain":"ev"`) || !strings.Contains(b, `"source":"garage"`) || !strings.Contains(b, `"engine":false`) {
		t.Fatalf("garage record: %s", b)
	}
	srv.Cfg.Set(settings.PowertrainKey, "phev")
	if _, b = do(t, admin, "GET", ts.URL+"/api/vehicle", "", nil); !strings.Contains(b, `"powertrain":"phev"`) || !strings.Contains(b, `"source":"setting"`) {
		t.Fatalf("the admin still wins: %s", b)
	}
}

func TestPollerLearnsTheEngineTypeFromTheGarage(t *testing.T) {
	_, srv := healthServer(t, fakeProv{}, false, false)
	srv.P.LearnCaps(ford.ParseState([]byte(`{"garage":{"engineType":"PHEV"}}`), time.Now()))
	if p, src := srv.powertrain(); p != ford.PHEV || src != "garage" {
		t.Fatalf("got %q from %s", p, src)
	}
}
