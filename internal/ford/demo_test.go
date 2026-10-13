package ford

import (
	"context"
	"testing"
)

func TestEachDemoCarLooksLikeItsType(t *testing.T) {
	for kind, want := range map[string]Powertrain{"": EV, "ev": EV, "gas": Gas, "hev": HEV, "phev": PHEV, "nonsense": EV} {
		d := NewDemoKind(kind)
		var caps Caps
		var engine string
		var plugged, fuelSeen, evSeen bool
		for i := 0; i < 24; i++ {
			s, err := d.Fetch(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			caps = caps.Or(CapsOf(s.Extra))
			engine = EngineTypeOf(s.Extra)
			plugged = plugged || s.Plug == "CONNECTED"
			fuelSeen = fuelSeen || s.FuelPct != nil
			evSeen = evSeen || s.SoCPct != nil
			if s.OutsideC == nil || *s.OutsideC != 18 {
				t.Fatalf("%s: outside temperature must come from outsideTemperature, not ambientTemp: %v", kind, s.OutsideC)
			}
		}
		if got := FromEngineType(engine); got != want {
			t.Errorf("%q: garage says %q -> %q, want %q", kind, engine, got, want)
		}
		if got := caps.Powertrain(); got != want {
			t.Errorf("%q: metrics suggest %q, want %q", kind, got, want)
		}
		if plugged != want.CanPlug() || fuelSeen != (want == Gas || want == HEV || want == PHEV) || evSeen != want.HasBattery() {
			t.Errorf("%q: plugged %v fuel %v battery %v", kind, plugged, fuelSeen, evSeen)
		}
	}
}
