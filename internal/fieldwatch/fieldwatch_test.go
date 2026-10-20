package fieldwatch

import "testing"

func TestEachMetricIsCountedByWhatChangedNotByItsTimestamps(t *testing.T) {
	row := func(ts int64, speed float64, lock string, stamp string) Row {
		return Row{TS: ts, Flat: map[string]any{
			"telemetry.metrics.speed.value":                  speed,
			"telemetry.metrics.speed.updateTime":             stamp,
			"telemetry.metrics.doorLockStatus[0].value":      lock,
			"telemetry.metrics.doorLockStatus[0].updateTime": stamp,
			"telemetry.metrics.doorLockStatus[1].value":      lock,
			"telemetry.metrics.odometer.value":               100.0,
			"telemetry.metrics.odometer.updateTime":          stamp,
			"garage.make":                                    "F",
		}}
	}
	rows := []Row{row(300, 20, "LOCKED", "c"), row(100, 0, "LOCKED", "a"), row(200, 50, "UNLOCKED", "b")}
	got := map[string]Stat{}
	for _, s := range Analyze(rows) {
		got[s.Metric] = s
	}
	if len(got) != 3 {
		t.Fatalf("three metrics, and nothing from outside telemetry.metrics: %+v", got)
	}
	sp := got["speed"]
	if sp.Samples != 3 || sp.Changes != 2 || sp.LastChangeTS != 300 || sp.Last != "20" || !sp.Numeric || *sp.Min != 0 || *sp.Max != 50 {
		t.Fatalf("speed: %+v", sp)
	}
	lk := got["doorLockStatus"]
	if lk.Changes != 2 || lk.Parts != 2 || lk.Last != "LOCKED / LOCKED" || lk.LastChangeTS != 300 {
		t.Fatalf("a list changes when any part does: %+v", lk)
	}
	od := got["odometer"]
	if od.Changes != 0 || od.LastChangeTS != 0 || od.ReportedAt != "c" {
		t.Fatalf("a new timestamp is not a change: %+v", od)
	}
}

func TestPositionsAndIdentifiersAreListedButTheirValuesAreHidden(t *testing.T) {
	rows := []Row{{TS: 1, Flat: map[string]any{"telemetry.metrics.position.value.location.lat": 44.76, "telemetry.metrics.position.value.location.lon": -85.62, "telemetry.metrics.vin.value": "1FMABC"}}}
	for _, s := range Analyze(rows) {
		if s.Metric == "position" || s.Metric == "vin" {
			if s.Min != nil || s.Last == "" || s.Last[:6] != "hidden" {
				t.Fatalf("%s leaked: %+v", s.Metric, s)
			}
		}
	}
	if len(Analyze(rows)) != 2 {
		t.Fatal("both are still listed")
	}
}
