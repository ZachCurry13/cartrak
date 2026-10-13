package ford

import (
	"os"
	"testing"
	"time"
)

func caps(t *testing.T, json string) Caps { return CapsOf(Flatten([]byte(json))) }

func TestPowertrainIsWorkedOutFromTheMetricsACarReports(t *testing.T) {
	cases := []struct {
		name, json string
		want       Powertrain
	}{
		{"battery electric", `{"m":{"xevBatteryStateOfCharge":{"value":80},"xevBatteryRange":{"value":300},"xevPlugChargerStatus":{"value":"DISCONNECTED"}}}`, EV},
		{"gas", `{"m":{"fuelLevel":{"value":55},"oilLifeRemaining":{"value":40},"engineSpeed":{"value":0}}}`, Gas},
		{"plug-in hybrid", `{"m":{"xevBatteryStateOfCharge":{"value":50},"fuelLevel":{"value":30},"xevPlugChargerStatus":{"value":"CONNECTED"}}}`, PHEV},
		{"hybrid", `{"m":{"xevBatteryStateOfCharge":{"value":50},"fuelLevel":{"value":30}}}`, HEV},
		{"oil life and hybrid mode alone prove nothing", `{"m":{"oilLifeRemaining":{"value":100},"hybridVehicleModeStatus":{"value":"AUTO_CHARGE_DEPLETE"}}}`, Unknown},
		{"nothing useful", `{"m":{"speed":{"value":3}}}`, Unknown},
		{"an EV listing an empty tank is not a gas car", `{"m":{"xevBatteryStateOfCharge":{"value":80},"fuelLevel":{"value":0}}}`, EV},
	}
	for _, c := range cases {
		if got := caps(t, c.json).Powertrain(); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

func TestEngineTypeFromTheGarage(t *testing.T) {
	for in, want := range map[string]Powertrain{
		"BEV": EV, "bev": EV, "Electric": EV, "PHEV": PHEV, "Plug-in Hybrid": PHEV, "HEV": HEV, "MHEV": HEV, "Hybrid": HEV,
		"ICE": Gas, "Gasoline": Gas, "Diesel": Gas, "": Unknown, "something new": Unknown,
	} {
		if got := FromEngineType(in); got != want {
			t.Errorf("%q: got %q, want %q", in, got, want)
		}
	}
}

// bev is an anonymised copy of the layout a real battery-electric car returned.
func bev(t *testing.T) []byte {
	b, err := os.ReadFile("testdata/bev_response.json")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestRealBatteryCarIsNotTakenForAHybrid(t *testing.T) {
	// This car lists oil life (100), hybrid mode, engine speed and coolant even though it has no engine.
	f := Flatten(bev(t))
	if got := CapsOf(f).Powertrain(); got != EV {
		t.Fatalf("from its metrics: %q", got)
	}
	if et := EngineTypeOf(f); et != "BEV" || FromEngineType(et) != EV {
		t.Fatalf("the garage says %q", et)
	}
}

func TestParseRealBatteryCarResponse(t *testing.T) {
	s := ParseState(bev(t), time.Now())
	near := func(name string, got *float64, want float64) {
		if got == nil || *got < want-0.001 || *got > want+0.001 {
			t.Errorf("%s = %v, want %v", name, got, want)
		}
	}
	near("outside temperature (not the 0 from ambientTemp)", s.OutsideC, 15.75)
	near("12 V battery", s.Battery12V, 12.0625)
	near("charge", s.SoCPct, 91.5)
	near("range", s.RangeKm, 355)
	near("odometer", s.OdometerKm, 50000)
	near("traction battery volts", s.HVVolts, 379)
	near("capacity", s.CapacityKwh, 88.45)
	near("oil life is parsed but a battery car never shows it", s.OilLifePct, 100)
	for i, want := range [4]float64{288, 290, 286, 292} {
		near("tire", s.Tires[i], want)
	}
	if s.Locked == nil || !*s.Locked || s.Ignition != "OFF" || s.Plug != "DISCONNECTED" {
		t.Errorf("lock %v ignition %q plug %q", s.Locked, s.Ignition, s.Plug)
	}
}
