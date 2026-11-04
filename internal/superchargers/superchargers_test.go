package superchargers

import (
	"strings"
	"testing"
)

const sample = `[
{"name":"A, VA","status":"OPEN","gps":{"latitude":39,"longitude":-77},"stallCount":8,"powerKilowatt":250,"otherEVs":true,"plugs":{"nacs":8}},
{"name":"Tesla only","status":"OPEN","gps":{"latitude":40,"longitude":-77},"otherEVs":false,"plugs":{"nacs":8}},
{"name":"Planned","status":"PLAN","gps":{"latitude":41,"longitude":-77},"otherEVs":true,"plugs":{"nacs":8}},
{"name":"Dock","status":"OPEN","gps":{"latitude":42,"longitude":-77},"powerKilowatt":150,"otherEVs":true,"plugs":{"multi":4}},
{"name":"No plugs","status":"OPEN","gps":{"latitude":43,"longitude":-77},"otherEVs":true,"plugs":{"tpc":4}}]`

func TestParseKeepsOpenSitesOthersCanUse(t *testing.T) {
	s, err := Parse(strings.NewReader(sample))
	if err != nil || len(s) != 2 || s[0].Name != "A, VA" || !s[0].NACS || s[1].Name != "Dock" || !s[1].Dock || s[1].NACS {
		t.Fatalf("%v %+v", err, s)
	}
	if n := len(Usable(s, "nacs")); n != 1 {
		t.Errorf("nacs: %d", n)
	}
	if n := len(Usable(s, "dock")); n != 1 {
		t.Errorf("dock: %d", n)
	}
	if n := len(Usable(s, "both")); n != 2 {
		t.Errorf("both: %d", n)
	}
}
