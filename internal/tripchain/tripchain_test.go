package tripchain

import (
	"errors"
	"testing"
)

// a straight road north: one degree of latitude is about 111 km
func st(name string, lat float64) Station { return Station{Name: name, Lat: lat, Lon: -93, Kw: 250} }

func base() Params {
	return Params{Start: st("Home", 40), Dest: st("There", 48), StartPct: 90, ReservePct: 10, TargetPct: 90, CapacityKwh: 88, KwhPerKm: 0.2, CarKw: 150, MaxLegMin: 150, DayMin: 12 * 60}
}

func ladder(to float64) []Station {
	var s []Station
	for lat := 40.5; lat < to; lat += 0.5 {
		s = append(s, st("S", lat))
	}
	return s
}

func TestEveryLegInsideTheLimits(t *testing.T) {
	days, err := Plan(base(), ladder(48), nil)
	if err != nil || len(days) < 2 {
		t.Fatalf("%v %d", err, len(days))
	}
	for _, d := range days {
		for i := 1; i < len(d.Stops); i++ {
			km := haversineKm(d.Stops[i-1].Station, d.Stops[i].Station) * bend
			if km/105*60 > 150.01 {
				t.Fatalf("a leg of %.0f km takes more than 2.5 h", km)
			}
		}
		if d.DriveMin+d.ChargeMin > 12*60+0.01 {
			t.Fatalf("a day of %.0f min", d.DriveMin+d.ChargeMin)
		}
	}
	last := days[len(days)-1]
	if last.Stops[len(last.Stops)-1].Kind != "dest" || days[0].Stops[len(days[0].Stops)-1].Kind != "overnight" {
		t.Fatalf("%+v", days)
	}
}

func TestShortTripNeedsNoStop(t *testing.T) {
	p := base()
	p.Dest = st("Near", 41)
	days, err := Plan(p, ladder(48), nil)
	if err != nil || len(days) != 1 || len(days[0].Stops) != 2 {
		t.Fatalf("%v %+v", err, days)
	}
}

func TestGapWithoutAChargerFails(t *testing.T) {
	if _, err := Plan(base(), ladder(44), nil); !errors.Is(err, ErrNoChain) {
		t.Fatalf("want ErrNoChain, got %v", err)
	}
}

func TestStationsOffTheRouteAreIgnored(t *testing.T) {
	p := base()
	p.Dest = st("Near", 41.5)
	far := Station{Name: "Far east", Lat: 41, Lon: -80, Kw: 250}
	days, err := Plan(p, []Station{far}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range days[0].Stops {
		if s.Name == "Far east" {
			t.Fatal("went far off the road")
		}
	}
}

func TestABookedHotelEndsTheDay(t *testing.T) {
	p := base()
	p.Dest = st("There", 46)
	days, err := Plan(p, ladder(48), []Via{{Station: st("Hotel", 43), Night: true, Charge: true}})
	if err != nil || len(days) != 2 {
		t.Fatalf("%v %+v", err, days)
	}
	last := days[0].Stops[len(days[0].Stops)-1]
	if last.Name != "Hotel" || last.Kind != "overnight" || days[1].Stops[0].Name != "Hotel" || days[1].StartPct != p.TargetPct {
		t.Fatalf("%+v", days)
	}
}

func TestAPreferredNetworkWinsUnlessTheOtherSavesEnough(t *testing.T) {
	p := base()
	p.Dest = st("There", 43)
	tesla := Station{Name: "Tesla", Lat: 41.5, Lon: -93, Kw: 250}
	other := Station{Name: "Other", Lat: 41.5, Lon: -93, Kw: 250, Penalty: 10}
	days, err := Plan(p, []Station{other, tesla}, nil)
	if err != nil || days[0].Stops[1].Name != "Tesla" {
		t.Fatalf("equal chargers: the one without a penalty wins: %v %+v", err, days)
	}
}
