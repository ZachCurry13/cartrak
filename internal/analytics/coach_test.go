package analytics

import (
	"strings"
	"testing"
)

func TestTheCoachOnlySpeaksWhenTheNumbersAreClear(t *testing.T) {
	var cs []CoachCharge
	for i := int64(0); i < 8; i++ { // home charges to 100%, followed by short drives, mostly in the daytime
		s := i * 86400
		cs = append(cs, CoachCharge{Start: s, End: s + 3*3600, Kwh: 30, WallKwh: 33, Where: "home", EndSoC: fp(100), StartSoC: fp(60), NextUsePct: fp(20), CheapShare: 0.2})
	}
	for i := int64(0); i < 4; i++ { // fast charges: two at a slow place
		place, mins := "Somewhere", int64(30)
		if i < 2 {
			place, mins = "Slow Stop", 90
		}
		cs = append(cs, CoachCharge{Start: 1e6 + i*1e4, End: 1e6 + i*1e4 + mins*60, Kwh: 45, Where: "public", Place: place, PeakKw: 150, Cost: fp(22), StartSoC: fp(8)})
	}
	tips := Coach(cs, Prices{Home: 0.16, Cheap: 0.08, Currency: "$"}, 60)
	got := map[string]Tip{}
	for _, x := range tips {
		got[x.Key] = x
	}
	if _, ok := got["full"]; !ok {
		t.Fatalf("charging to full: %+v", tips)
	}
	if n := got["night"]; n.Worth < 8.4 || n.Worth > 8.5 { // 8 x 33 x 0.8 = 211 kWh in the day, x 0.08 a kWh, over two months: 8.45 a month
		t.Fatalf("night savings: %+v", n)
	}
	if s := got["slow"]; !strings.Contains(s.Title, "Slow Stop") {
		t.Fatalf("slow charger: %+v", tips)
	}
	if _, ok := got["low"]; !ok {
		t.Fatal("arriving low four times")
	}
	if _, ok := got["public"]; !ok {
		t.Fatal("public at three times the home price")
	}
	if tips[0].Key != "public" {
		t.Fatalf("the most valuable first: %+v", tips[0])
	}
	if len(Coach(cs[:3], Prices{}, 60)) != 0 {
		t.Fatal("too little to go on")
	}
}
