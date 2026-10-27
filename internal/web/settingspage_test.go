package web

import (
	"strings"
	"testing"

	"github.com/ZachCurry13/cartrak/internal/settings"
)

// The Charging settings page lists its settings by key. A charging setting missing from that list is saved by nobody and shown
// nowhere, which is how the EIA key once went missing from its own page.
func TestEveryChargingSettingHasAPlaceOnTheChargingSettingsPage(t *testing.T) {
	page, err := Files.ReadFile("static/js/pages/chargingsettings.js")
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range settings.Defs {
		if settings.GroupOf(d) == "charging" && !strings.Contains(string(page), "'"+d.Key+"'") {
			t.Errorf("%s (%s) is a charging setting but the page never shows it", d.Key, d.Label)
		}
	}
}
