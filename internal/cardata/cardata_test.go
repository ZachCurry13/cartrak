package cardata

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
)

var reading = time.Date(2026, 10, 12, 22, 35, 0, 0, time.UTC)

func fixture(t *testing.T) map[string]any {
	b, err := os.ReadFile("../ford/testdata/bev_response.json")
	if err != nil {
		t.Fatal(err)
	}
	return ford.Flatten(b)
}

func tabOf(r Report, id string) *Tab {
	for i := range r.Tabs {
		if r.Tabs[i].ID == id {
			return &r.Tabs[i]
		}
	}
	return nil
}

func rowOf(t *Tab, label string) *Row {
	if t == nil {
		return nil
	}
	for _, g := range t.Groups {
		for i := range g.Rows {
			if g.Rows[i].Label == label {
				return &g.Rows[i]
			}
		}
	}
	return nil
}

func groupOf(t *Tab, title string) *Group {
	if t == nil {
		return nil
	}
	for i := range t.Groups {
		if t.Groups[i].Title == title {
			return &t.Groups[i]
		}
	}
	return nil
}

func TestARealBatteryCarGetsEveryTabInPlainWords(t *testing.T) {
	r := Build(fixture(t), reading, ford.EV)
	ids := []string{}
	for _, tb := range r.Tabs {
		ids = append(ids, tb.ID)
	}
	if strings.Join(ids, ",") != "summary,battery,driving,body,settings,car" {
		t.Fatalf("tabs: %v", ids)
	}
	b := tabOf(r, "battery")
	if groupOf(b, "Fuel and engine") != nil {
		t.Fatal("a battery car has no fuel tank, even though Ford lists an oil life of 100 for it")
	}
	cases := map[string]float64{"Charge shown on the dash": 91.5, "Actual charge": 86.54, "Energy left": 75.3, "Battery size (estimated)": 88.45, "Time to full": 844}
	for label, want := range cases {
		if x := rowOf(b, label); x == nil || x.Num == nil || *x.Num != want || x.Help == "" || x.At == "" {
			t.Errorf("%s: %+v", label, x)
		}
	}
	if x := rowOf(b, "Time to full"); x.Kind != "minutes" {
		t.Errorf("the page needs to know 844 is minutes: %q", x.Kind)
	}
	if x := rowOf(b, "Station type"); x == nil || x.Text != "Ac basic" && x.Text != "AC basic" && !strings.Contains(strings.ToLower(x.Text), "ac basic") {
		t.Errorf("station type: %+v", x)
	}
	if x := rowOf(tabOf(r, "summary"), "State"); x == nil || x.Text != "Parked" {
		t.Errorf("state: %+v", x)
	}
	if x := rowOf(tabOf(r, "summary"), "Doors and locks"); x == nil || x.Text != "Open: Tailgate" {
		t.Errorf("the open tailgate is called out: %+v", x)
	}
}

func TestBodySettingsAndCarTabsReadTheRealLists(t *testing.T) {
	r := Build(fixture(t), reading, ford.EV)
	body := tabOf(r, "body")
	if x := rowOf(body, "Driver front window"); x == nil || x.Text != "Open up to 40%" {
		t.Errorf("window: %+v", x)
	}
	if x := rowOf(body, "Passenger front window"); x == nil || x.Text != "Closed" {
		t.Errorf("window: %+v", x)
	}
	if x := rowOf(body, "Passenger seat belt"); x == nil || x.Text != "Unbuckled" {
		t.Errorf("seat belt: %+v", x)
	}
	if x := rowOf(body, "Alarm"); x == nil || x.Text != "Armed" {
		t.Errorf("alarm: %+v", x)
	}
	if x := rowOf(body, "Front left"); x == nil || x.Kind != "kpa" || x.Num == nil || *x.Num != 288 || x.Text != "Normal" {
		t.Errorf("a tire carries its pressure and the car's verdict: %+v", x)
	}
	if x := rowOf(body, "Recommended (front)"); x == nil || x.Num == nil || *x.Num < 268 || *x.Num > 270 {
		t.Errorf("sticker pressure: %+v", x)
	}
	set := tabOf(r, "settings")
	if x := rowOf(set, "Update time"); x == nil || x.Text != "Every day at 1:00 AM" {
		t.Errorf("software update window: %+v", x)
	}
	if x := rowOf(set, "Departure times"); x == nil || !strings.Contains(x.Text, "1 is turned on of 4 slots") {
		t.Errorf("departure summary: %+v", x)
	}
	if x := rowOf(set, "Monday"); x == nil || x.Text != "7:30 AM (on)" {
		t.Errorf("a departure time that is on is listed: %+v", x)
	}
	if x := rowOf(set, "Remote start run time"); x == nil || x.Kind != "duration" || *x.Num != 600 {
		t.Errorf("remote start run time: %+v", x)
	}
	car := tabOf(r, "car")
	if x := rowOf(car, "Model"); x == nil || x.Text != "2024 Mustang Mach-E" {
		t.Errorf("model: %+v", x)
	}
	if x := rowOf(car, "Make"); x == nil || x.Text != "Ford" {
		t.Errorf("make: %+v", x)
	}
	if x := rowOf(car, "Treated by CarTrak as"); x == nil || x.Text != "Electric" {
		t.Errorf("type: %+v", x)
	}
}

func TestDrivingTabConvertsSpeedAndNeverShowsAPlace(t *testing.T) {
	f := fixture(t)
	for k := range f {
		if strings.HasSuffix(k, "speed.value") {
			f[k] = 10.0 // metres per second
		}
	}
	r := Build(f, reading, ford.EV)
	if x := rowOf(tabOf(r, "driving"), "Speed"); x == nil || x.Kind != "kph" || *x.Num != 36 {
		t.Errorf("10 m/s is 36 km/h: %+v", x)
	}
	if x := rowOf(tabOf(r, "driving"), "Heading"); x == nil || *x.Num != 48 || x.Kind != "deg" {
		t.Errorf("heading: %+v", x)
	}
	b, _ := json.Marshal(r)
	for _, secret := range []string{"TESTVIN", `"position"`, "latitude", `"lat"`, "location"} {
		if strings.Contains(string(b), secret) {
			t.Fatalf("the Car data page must never carry %q: %s", secret, b)
		}
	}
}

func TestFreshnessIsRelativeToTheReadingNotToNow(t *testing.T) {
	r := Build(fixture(t), reading, ford.EV)
	x := rowOf(tabOf(r, "summary"), "Values updated recently")
	if x == nil || !strings.Contains(x.Text, "within 2 hours") || !strings.Contains(x.Text, "the oldest is from") {
		t.Fatalf("freshness: %+v", x)
	}
	// Looking at a reading from long ago: values are judged against that moment.
	early := Build(fixture(t), time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC), ford.EV)
	if y := rowOf(tabOf(early, "summary"), "Values updated recently"); y == nil {
		t.Fatal("still describable")
	}
}

func TestAGasCarShowsFuelAndOilButNoBatteries(t *testing.T) {
	f := ford.Flatten([]byte(`{"telemetry":{"metrics":{"fuelLevel":{"value":55},"fuelRange":{"value":400},"oilLifeRemaining":{"value":40},
	  "engineCoolantTemp":{"value":90},"batteryVoltage":{"value":12.7},"odometer":{"value":1000}}}}`))
	r := Build(f, reading, ford.Gas)
	b := tabOf(r, "battery")
	if groupOf(b, "Main battery") != nil || groupOf(b, "Charging") != nil {
		t.Fatal("a gas car has no main battery or charging")
	}
	if x := rowOf(b, "Oil life left"); x == nil || *x.Num != 40 {
		t.Errorf("oil: %+v", x)
	}
	if x := rowOf(tabOf(r, "summary"), "Fuel"); x == nil || *x.Num != 55 {
		t.Errorf("fuel on the summary: %+v", x)
	}
	// An unknown type must not show oil just because a battery car lists it.
	u := Build(ford.Flatten([]byte(`{"m":{"oilLifeRemaining":{"value":100},"xevBatteryStateOfCharge":{"value":80}}}`)), reading, ford.Unknown)
	if groupOf(tabOf(u, "battery"), "Fuel and engine") != nil {
		t.Fatal("oil life alone does not make an engine")
	}
}

func TestEveryRowIsWellFormed(t *testing.T) {
	kinds := map[string]bool{"volts": true, "pct": true, "kwh": true, "amps": true, "tempC": true, "km": true, "kpa": true, "kph": true, "seconds": true,
		"duration": true, "minutes": true, "nm": true, "accel": true, "degps": true, "deg": true, "rpm": true, "text": true}
	for _, tb := range Build(fixture(t), reading, ford.EV).Tabs {
		if tb.Title == "" || len(tb.Groups) == 0 {
			t.Errorf("tab %q is empty", tb.ID)
		}
		for _, g := range tb.Groups {
			for _, r := range g.Rows {
				if r.Label == "" || !kinds[r.Kind] || (r.Kind != "text" && r.Num == nil) {
					t.Errorf("%s / %s: malformed row %+v", tb.ID, g.Title, r)
				}
			}
		}
	}
}

func TestDoorRowsSayWhenTheCarLastReportedThem(t *testing.T) {
	p := "telemetry.metrics.doorStatus"
	f := map[string]any{
		p + "[0].value": "AJAR", p + "[0].vehicleDoor": "UNSPECIFIED_FRONT", p + "[0].vehicleSide": "PASSENGER", p + "[0].updateTime": "2026-10-20T14:29:55Z",
		p + "[1].value": "CLOSED", p + "[1].vehicleDoor": "UNSPECIFIED_FRONT", p + "[1].vehicleSide": "DRIVER", p + "[1].updateTime": "2026-10-20T14:29:55Z",
	}
	body := tabOf(Build(f, reading, ford.EV), "body")
	if x := rowOf(body, "Passenger front"); x == nil || x.Text != "Ajar" || x.At != "2026-10-20T14:29:55Z" {
		t.Errorf("a door row carries the car's own report time: %+v", x)
	}
}
