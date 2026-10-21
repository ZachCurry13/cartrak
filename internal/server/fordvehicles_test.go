package server

import (
	"strings"
	"testing"

	"github.com/ZachCurry13/cartrak/internal/ford"
)

type twoVehicles struct{ fakeLinker }

func (twoVehicles) Vehicles() []ford.VehicleChoice {
	return []ford.VehicleChoice{{VIN: "1FTVW1EL0PWG00001", Label: "2023 F-150 Lightning"}, {VIN: "1FMCU9J90NUA00002", Label: "Vehicle ending A00002"}}
}

func TestTheOwnerChoosesWhichVehicleACarFollows(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	srv.Link = twoVehicles{}
	_ = srv.S.SetKV(ford.SeenVINKey, "1FTVW1EL0PWG00001")
	_ = srv.S.SetKV(ford.EngineKV, "BEV") // learned from the F-150
	_ = srv.S.SetKV(ford.CapsKV, "ev,plug")
	admin := adminClient(t, ts.URL)
	_, body := do(t, admin, "GET", ts.URL+"/api/ford/vehicles", "", nil)
	if !strings.Contains(body, `"label":"2023 F-150 Lightning","this":true`) || !strings.Contains(body, `"label":"Vehicle ending A00002","this":false`) {
		t.Fatalf("both listed, the followed one marked: %s", body)
	}
	if code, _ := do(t, admin, "POST", ts.URL+"/api/ford/vehicle", `{"vin":"1FMCU9J90NUA00002"}`, xh); code != 200 {
		t.Fatal("choose")
	}
	if srv.Cfg.Value("FORD_VIN") != "1FMCU9J90NUA00002" || srv.S.GetKV(ford.SeenVINKey) != "1FMCU9J90NUA00002" {
		t.Fatal("the choice is kept")
	}
	if srv.S.GetKV(ford.EngineKV) != "" || srv.S.GetKV(ford.CapsKV) != "" {
		t.Fatal("the other vehicle's type is forgotten, to be learned from this one")
	}
	if code, _ := do(t, admin, "POST", ts.URL+"/api/ford/vehicle", `{"vin":"NOT-ON-THE-ACCOUNT"}`, xh); code != 404 {
		t.Fatal("only a vehicle on the account")
	}
}
