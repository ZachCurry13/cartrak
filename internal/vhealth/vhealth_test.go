package vhealth

import (
	"os"
	"testing"

	"github.com/ZachCurry13/cartrak/internal/ford"
)

func flat(t *testing.T, json string) map[string]any { return ford.Flatten([]byte(json)) }

func section(r Report, id string) *Section {
	for i := range r.Sections {
		if r.Sections[i].ID == id {
			return &r.Sections[i]
		}
	}
	return nil
}

func row(s *Section, label string) *Row {
	if s == nil {
		return nil
	}
	for i := range s.Rows {
		if s.Rows[i].Label == label {
			return &s.Rows[i]
		}
	}
	return nil
}

func TestRealBatteryCarShowsVoltagesAndBatteryButNoOil(t *testing.T) {
	raw, err := os.ReadFile("../ford/testdata/bev_response.json")
	if err != nil {
		t.Fatal(err)
	}
	r := Build(ford.Flatten(raw), ford.EV, nil)

	if section(r, "engine") != nil {
		t.Fatal("a battery car must not get an oil section, even though Ford lists oil life 100 for it")
	}
	v := row(section(r, "12v"), "Voltage")
	if v == nil || v.Num == nil || *v.Num != 12.0625 || v.State != "warn" || v.At == "" {
		t.Fatalf("12 V voltage: %+v", v)
	}
	if c := row(section(r, "12v"), "Charge"); c == nil || *c.Num != 65 || c.State != "warn" {
		t.Fatalf("12 V charge: %+v", c)
	}
	tr := section(r, "traction")
	for label, want := range map[string]float64{"Voltage": 379, "Estimated capacity": 88.45, "Temperature": 13, "Energy left": 75.3, "Charge shown on the dash": 91.5, "Actual charge": 86.54} {
		if x := row(tr, label); x == nil || x.Num == nil || *x.Num != want {
			t.Errorf("main battery %q: %+v", label, x)
		}
	}
	if st := row(tr, "Battery status"); st == nil || st.Text != "Normal" || st.State != "ok" {
		t.Errorf("battery status: %+v", st)
	}
	if x := row(tr, "Estimated capacity"); x.At != "2026-10-09T14:17:32Z" {
		t.Errorf("each value carries when Ford last updated it: %q", x.At)
	}
	if w := section(r, "warnings"); row(w, "Trouble codes") == nil || row(w, "Vehicle alerts") == nil || row(w, "Warning lights") == nil {
		t.Fatalf("with nothing wrong the page says so: %+v", w)
	}
	for _, x := range section(r, "tires").Rows {
		if x.Label != "Tire monitor" && x.State != "ok" {
			t.Errorf("tire %+v", x)
		}
	}
	if r.Overall != "warn" {
		t.Fatalf("a lowish 12 V battery makes the overall state a warning: %s", r.Overall)
	}
}

func TestGasCarShowsOilAndFuelButNoMainBattery(t *testing.T) {
	f := flat(t, `{"telemetry":{"metrics":{"oilLifeRemaining":{"value":12},"fuelLevel":{"value":8},"fuelRange":{"value":60},
	  "engineCoolantTemp":{"value":90},"batteryVoltage":{"value":12.7},"hybridVehicleModeStatus":{"value":"AUTO"}}}}`)
	r := Build(f, ford.Gas, nil)
	if section(r, "traction") != nil {
		t.Fatal("a gas car has no main battery section")
	}
	e := section(r, "engine")
	if o := row(e, "Oil life left"); o == nil || o.State != "warn" || o.Note == "" {
		t.Fatalf("oil at 12%% should say to plan a change: %+v", o)
	}
	if fl := row(e, "Fuel level"); fl == nil || fl.State != "warn" {
		t.Fatalf("fuel: %+v", fl)
	}
	if row(e, "Hybrid mode") != nil {
		t.Fatal("hybrid mode is only for hybrids")
	}
	if v := row(section(r, "12v"), "Voltage"); v == nil || v.State != "ok" {
		t.Fatalf("12.7 V is healthy: %+v", v)
	}
	f2 := flat(t, `{"m":{"oilLifeRemaining":{"value":4}}}`)
	if o := row(section(Build(f2, ford.Gas, nil), "engine"), "Oil life left"); o.State != "bad" {
		t.Fatalf("oil at 4%%: %+v", o)
	}
}

func TestHybridShowsBothBatteryAndOil(t *testing.T) {
	f := flat(t, `{"m":{"xevBatteryStateOfCharge":{"value":55},"oilLifeRemaining":{"value":70},"fuelLevel":{"value":40},"hybridVehicleModeStatus":{"value":"AUTO_CHARGE_DEPLETE"}}}`)
	r := Build(f, ford.PHEV, nil)
	if section(r, "traction") == nil || section(r, "engine") == nil || row(section(r, "engine"), "Hybrid mode").Text != "Auto charge deplete" {
		t.Fatalf("plug-in hybrid sections: %+v", r.Sections)
	}
}

func TestUnknownTypeNeverShowsOilUnlessThereIsFuel(t *testing.T) {
	f := flat(t, `{"m":{"oilLifeRemaining":{"value":100},"xevBatteryStateOfCharge":{"value":80}}}`)
	if section(Build(f, ford.Unknown, nil), "engine") != nil {
		t.Fatal("oil life alone must not create an engine section before the type is known")
	}
	f2 := flat(t, `{"m":{"oilLifeRemaining":{"value":100},"fuelLevel":{"value":50}}}`)
	if section(Build(f2, ford.Unknown, nil), "engine") == nil {
		t.Fatal("a fuel level proves there is an engine")
	}
}

func TestCodesAlertsAndWarningLightsAreListed(t *testing.T) {
	f := flat(t, `{"telemetry":{"metrics":{"diagnosticTroubleCode":[{"value":"P0A80","system":"POWERTRAIN"},{"value":"U0100","system":"NETWORK"}],
	  "indicators":{"brakePadWearWarning":{"value":true},"tirePressureWarning":{"value":false}}}},
	  "alerts":{"VehicleAlertList":[{"StatusCode":4040,"StatusDesc":"No Data"},{"StatusCode":1,"StatusDesc":"Service required"}]}}`)
	w := section(Build(f, ford.EV, nil), "warnings")
	var codes, alerts, lights []string
	for _, x := range w.Rows {
		switch x.Label {
		case "Trouble code":
			codes = append(codes, x.Text+"/"+x.Note)
		case "Ford alert":
			alerts = append(alerts, x.Text)
		case "Warning light":
			lights = append(lights, x.Text)
		}
	}
	if len(codes) != 2 || codes[0] != "P0A80/Powertrain system" || len(alerts) != 1 || alerts[0] != "Service required" || len(lights) != 1 || lights[0] != "Brake pad wear warning" {
		t.Fatalf("codes %v alerts %v lights %v", codes, alerts, lights)
	}
}

func TestLowTirePressureIsFlaggedAndTrendsNeedTwoDays(t *testing.T) {
	f := flat(t, `{"m":{"tirePressure":[{"value":200,"vehicleWheel":"FRONT_LEFT","wheelPlacardFront":269}],"tirePressureStatus":[{"value":"LOW","vehicleWheel":"FRONT_LEFT"}]}}`)
	if x := row(section(Build(f, ford.EV, nil), "tires"), "Front left"); x == nil || x.State != "warn" {
		t.Fatalf("tire: %+v", x)
	}
	one := Trends{"battery12v": {{TS: 1, Avg: 12.5}}}
	two := Trends{"battery12v": {{TS: 1, Avg: 12.5}, {TS: 86401, Avg: 12.4}}}
	g := flat(t, `{"m":{"batteryVoltage":{"value":12.5}}}`)
	if s := section(Build(g, ford.EV, one), "12v"); len(s.Series) != 0 {
		t.Fatal("one day is not a trend")
	}
	if s := section(Build(g, ford.EV, two), "12v"); len(s.Series) != 1 || s.Series[0].Name != "battery12v" {
		t.Fatalf("two days make a trend: %+v", s.Series)
	}
}
