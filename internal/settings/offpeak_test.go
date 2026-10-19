package settings

import "testing"

func TestParseWindow(t *testing.T) {
	if f, to, ok := ParseWindow("22:00-06:30"); !ok || f != 22*60 || to != 6*60+30 {
		t.Fatalf("%d %d %v", f, to, ok)
	}
	for _, bad := range []string{"", "22:00", "22-06", "25:00-06:00", "22:00-22:00", "22:0-06:00", "a-b"} {
		if _, _, ok := ParseWindow(bad); ok {
			t.Errorf("accepted %q", bad)
		}
	}
}

func TestTheWindLookupIsOffUnlessSwitchedOn(t *testing.T) {
	if err := validators[WeatherKey]("on"); err != nil {
		t.Fatal(err)
	}
	if err := validators[WeatherKey](""); err != nil {
		t.Fatal(err)
	}
	if validators[WeatherKey]("maybe") == nil {
		t.Fatal("only off or on")
	}
}
