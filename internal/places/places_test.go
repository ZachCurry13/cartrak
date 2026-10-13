package places

import (
	"testing"

	"github.com/ZachCurry13/cartrak/internal/store"
)

func TestAPositionMatchesTheClosestNamedPlaceItIsInside(t *testing.T) {
	home := &store.NamedPlace{ID: 1, Name: "Home", Lat: 40, Lon: -75, RadiusM: 150}
	office := &store.NamedPlace{ID: 2, Name: "Office", Lat: 40.05, Lon: -75, RadiusM: 300}
	all := []*store.NamedPlace{home, office}
	if p := Match(40.0005, -75, all); p == nil || p.Name != "Home" { // 55 m from home
		t.Fatalf("%+v", p)
	}
	if Match(40.01, -75, all) != nil { // 1.1 km from home
		t.Fatal("too far from both")
	}
	lat, lon := 40.0499, -75.0
	if Name(&lat, &lon, all) != "Office" || Name(nil, nil, all) != "" {
		t.Fatal("names")
	}
	overlap := []*store.NamedPlace{{Name: "Wide", Lat: 40, Lon: -75, RadiusM: 500}, {Name: "Close", Lat: 40.0002, Lon: -75, RadiusM: 100}}
	if p := Match(40.0002, -75, overlap); p.Name != "Close" {
		t.Fatalf("the closest wins: %s", p.Name)
	}
}
