package ford

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestOneVehicleIsKeptWhenTheAccountAuthorizedSeveral(t *testing.T) {
	two := []byte(`{"vehicles":[{"vin":"1FTVW1EL0PWG00001","modelName":"F-150 Lightning","modelYear":2023,"metrics":{"odometer":{"value":100}}},{"vin":"1FMCU9J90NUA00002","metrics":{"odometer":{"value":200}}}]}`)
	out, list, chosen := pickVehicle(two, "", nil)
	if len(list) != 2 || chosen != "1FTVW1EL0PWG00001" || list[0].Label != "2023 F-150 Lightning" || list[1].Label != "Vehicle ending A00002" {
		t.Fatalf("first by default: %q %+v", chosen, list)
	}
	if strings.Contains(string(out), "00002") {
		t.Fatal("the other vehicle is left out")
	}
	out, _, chosen = pickVehicle(two, "1fmcu9j90nua00002", nil)
	if chosen != "1FMCU9J90NUA00002" || strings.Contains(string(out), "00001") || !strings.Contains(string(out), `"value":200`) {
		t.Fatalf("the chosen one: %s", out)
	}
	if _, _, chosen := pickVehicle(two, "", []string{"1ftvw1el0pwg00001"}); chosen != "1FMCU9J90NUA00002" {
		t.Fatalf("a car left unchosen takes the vehicle no other car follows: %q", chosen)
	}
	if _, _, chosen := pickVehicle(two, "1FTVW1EL0PWG00001", []string{"1FTVW1EL0PWG00001"}); chosen != "1FTVW1EL0PWG00001" {
		t.Fatal("a vehicle chosen for this car wins")
	}
	one := []byte(`{"vehicles":[{"vin":"X1","metrics":{}}]}`)
	if out, list, chosen := pickVehicle(one, "nope", nil); string(out) != string(one) || len(list) != 1 || chosen != "X1" {
		t.Fatal("a single vehicle is untouched")
	}
}

func TestACarKeepsItsVehicleWhenFordReordersTheList(t *testing.T) {
	const truck = `{"vin":"VIN-TRUCK","metrics":[{"name":"ODOMETER","value":100}]}`
	const suv = `{"vin":"VIN-SUV","metrics":[{"name":"ODOMETER","value":200}]}`
	f := newFake2(t)
	odo := func(c *Connect2) float64 {
		c.minGap = time.Nanosecond // ask Ford every time
		time.Sleep(2 * time.Millisecond)
		s, err := c.Fetch(context.Background())
		if err != nil || s.OdometerKm == nil {
			t.Fatalf("fetch: %v", err)
		}
		return *s.OdometerKm
	}
	kv := memKV{refreshKey: "R0"}
	c := f.client(kv)
	f.telemetry = `{"vehicles":[` + truck + `,` + suv + `]}`
	if odo(c) != 100 {
		t.Fatal("first reading: the first vehicle")
	}
	f.telemetry = `{"vehicles":[` + suv + `,` + truck + `]}` // Ford's order changed
	if odo(c) != 100 || kv[SeenVINKey] != "VIN-TRUCK" || atomic.LoadInt32(&f.telemetryHit) != 2 {
		t.Fatal("it keeps following the truck")
	}
	// a second car on the same account, not told which vehicle is its own: it takes the one the first car does not follow
	kv2 := memKV{refreshKey: "R0"}
	c2 := NewConnect2(func() Connect2Opts {
		return Connect2Opts{ClientID: "cid", ClientSecret: "sec", APIBase: f.srv.URL, TokenURL: f.srv.URL + "/token",
			Taken: func() []string { return []string{"VIN-TRUCK"} }}
	}, kv2)
	f.telemetry = `{"vehicles":[` + truck + `,` + suv + `]}`
	if odo(c2) != 200 {
		t.Fatal("the second car follows the SUV")
	}
}
