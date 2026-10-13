package server

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/cardata"
	"github.com/ZachCurry13/cartrak/internal/ford"
)

func TestTheCarCanBeBrowsedAtAnyMomentInHistory(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	raw, err := os.ReadFile("../ford/testdata/bev_response.json")
	if err != nil {
		t.Fatal(err)
	}
	base := time.Now().Add(-6 * time.Hour).Truncate(time.Minute)
	for i, soc := range []float64{80, 70, 60} {
		st := ford.ParseState(raw, base.Add(time.Duration(i)*time.Hour))
		st.SoCPct = &soc
		st.Extra["telemetry.metrics.xevBatteryStateOfCharge.value"] = soc
		if _, err := srv.S.SaveState("t", st); err != nil {
			t.Fatal(err)
		}
	}
	srv.P.LearnCaps(ford.ParseState(raw, base))

	get := func(q string) (map[string]any, string) {
		_, body := do(t, admin, "GET", ts.URL+"/api/car-data"+q, "", nil)
		var m map[string]any
		json.Unmarshal([]byte(body), &m)
		return m, body
	}
	charge := func(m map[string]any) float64 {
		var rep cardata.Report
		b, _ := json.Marshal(m["report"])
		json.Unmarshal(b, &rep)
		for _, tb := range rep.Tabs {
			if tb.ID == "battery" {
				for _, g := range tb.Groups {
					for _, r := range g.Rows {
						if r.Label == "Charge shown on the dash" {
							return *r.Num
						}
					}
				}
			}
		}
		return -1
	}

	now, body := get("")
	if now["ready"] != true || now["latest"] != true || charge(now) != 60 || now["model"] != "2024 Mustang Mach-E" {
		t.Fatalf("by default the latest reading: %s", body)
	}
	earlier, _ := get("?at=" + strconv.FormatInt(base.Add(90*time.Minute).Unix(), 10))
	if charge(earlier) != 70 || earlier["latest"] == true || earlier["prev"] == float64(0) || earlier["next"] == float64(0) {
		t.Fatalf("between readings the earlier one applies, and it can step both ways: %v", earlier)
	}
	first, _ := get("?at=" + strconv.FormatInt(base.Unix(), 10))
	if charge(first) != 80 || first["prev"] != float64(0) {
		t.Fatalf("the first reading has nothing before it: %v", first)
	}
	before, _ := get("?at=" + strconv.FormatInt(base.Add(-time.Hour).Unix(), 10))
	if before["ready"] != false || before["first"] != float64(base.Unix()) {
		t.Fatalf("before any reading: %v", before)
	}
	if strings.Contains(body, "TESTVIN") {
		t.Fatal("the page must never carry the VIN")
	}

	_, rawBody := do(t, admin, "GET", ts.URL+"/api/car-data/raw", "", nil)
	if !strings.Contains(rawBody, "telemetry.metrics.xevBatteryVoltage.value") {
		t.Fatalf("the All fields tab gets every value: %.200s", rawBody)
	}
	if code, _ := do(t, &http.Client{}, "GET", ts.URL+"/api/car-data", "", nil); code != 401 {
		t.Fatalf("anonymous: %d", code)
	}
}

func TestTheTimelineGivesOneDayFromTheSmallTable(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	day := time.Date(2026, 6, 10, 8, 0, 0, 0, time.UTC)
	for i := 0; i < 20; i++ {
		soc := 90 - float64(i)
		srv.S.SaveState("t", &ford.State{TS: day.Add(time.Duration(i) * 30 * time.Minute), SoCPct: &soc, Ignition: "OFF", Raw: []byte(`{}`)})
	}
	srv.S.SaveState("t", &ford.State{TS: day.AddDate(0, 0, 3), Raw: []byte(`{}`)}) // another day

	_, body := do(t, admin, "GET", ts.URL+"/api/timeline?day=2026-06-10", "", nil)
	var out struct {
		Points []struct {
			TS  int64    `json:"ts"`
			SoC *float64 `json:"soc"`
		} `json:"points"`
	}
	if err := json.Unmarshal([]byte(body), &out); err != nil || len(out.Points) != 20 || out.Points[0].SoC == nil || *out.Points[0].SoC != 90 {
		t.Fatalf("one day of readings: %d %s", len(out.Points), body)
	}
	for name, q := range map[string]string{"bad day": "?day=yesterday", "backwards": "?from=100&to=50", "too long": "?from=1&to=99999999999"} {
		if code, _ := do(t, admin, "GET", ts.URL+"/api/timeline"+q, "", nil); code != 400 {
			t.Errorf("%s should be refused, got %d", name, code)
		}
	}
}
