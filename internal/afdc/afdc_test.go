package afdc

import (
	"strings"
	"testing"
)

func TestParseKeepsCCSFastChargers(t *testing.T) {
	s, err := Parse(strings.NewReader(`{"fuel_stations":[
{"station_name":"Mall","latitude":44,"longitude":-90,"ev_dc_fast_num":6,"ev_connector_types":["J1772COMBO","CHADEMO"]},
{"station_name":"Level 2","latitude":45,"longitude":-90,"ev_dc_fast_num":0,"ev_connector_types":["J1772"]}]}`))
	if err != nil || len(s) != 1 || s[0].Name != "Electrify America: Mall" || !s[0].Dock || s[0].NACS || s[0].Net == "" {
		t.Fatalf("%v %+v", err, s)
	}
}
