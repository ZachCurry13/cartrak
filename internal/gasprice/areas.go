// Package gasprice looks up the weekly average retail price of regular gasoline from the US Energy Information
// Administration (EIA), for the state a charge happened in, or the region around it where the EIA publishes no state.
package gasprice

import "strings"

// Area is a place the EIA publishes a weekly gasoline price for.
type Area struct {
	ID    string // the EIA "duoarea" code
	Label string // how the app names it: "Minnesota average", "Midwest average"
}

// stateAreas are the states with a price of their own.
var stateAreas = map[string]Area{
	"CA": {"SCA", "California average"}, "CO": {"SCO", "Colorado average"}, "FL": {"SFL", "Florida average"},
	"MA": {"SMA", "Massachusetts average"}, "MN": {"SMN", "Minnesota average"}, "NY": {"SNY", "New York average"},
	"OH": {"SOH", "Ohio average"}, "TX": {"STX", "Texas average"}, "WA": {"SWA", "Washington average"},
}

// regionAreas are the regions (EIA's PADD sub-regions) every other state falls into.
var regionAreas = map[string]Area{
	"R1X": {"R1X", "New England average"}, "R1Y": {"R1Y", "Central Atlantic average"}, "R1Z": {"R1Z", "Lower Atlantic average"},
	"R20": {"R20", "Midwest average"}, "R30": {"R30", "Gulf Coast average"}, "R40": {"R40", "Rocky Mountain average"},
	"R50": {"R50", "West Coast average"},
}

// National is the fallback when a state is unknown.
var National = Area{"NUS", "US average"}

var stateRegion = func() map[string]string {
	m := map[string]string{}
	for region, states := range map[string]string{
		"R1X": "CT ME MA NH RI VT", "R1Y": "DE DC MD NJ NY PA", "R1Z": "FL GA NC SC VA WV",
		"R20": "IL IN IA KS KY MI MN MO NE ND OH OK SD TN WI", "R30": "AL AR LA MS NM TX",
		"R40": "CO ID MT UT WY", "R50": "AK AZ CA HI NV OR WA",
	} {
		for _, st := range strings.Fields(states) {
			m[st] = region
		}
	}
	return m
}()

// ForState is the area whose price applies in a state: its own when it has one, else its region, else the US average.
func ForState(code string) Area {
	code = strings.ToUpper(strings.TrimSpace(code))
	if a, ok := stateAreas[code]; ok {
		return a
	}
	if a, ok := regionAreas[stateRegion[code]]; ok {
		return a
	}
	return National
}
