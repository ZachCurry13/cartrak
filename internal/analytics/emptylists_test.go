package analytics

import (
	"encoding/json"
	"regexp"
	"testing"
	"time"
)

// Every report the Insights pages read must send lists as lists, even with nothing behind them: a null where a list is expected
// breaks the page. (The check is on the empty case, the one a new or quiet car meets.)
func TestReportsWithNothingBehindThemSendNoNullLists(t *testing.T) {
	loc := time.UTC
	reports := map[string]any{
		"driving":     Driving(nil, loc),
		"charging":    Charging(nil, loc),
		"battery":     Battery(nil, loc),
		"records":     Records(nil, nil, loc),
		"personality": Personalities(nil, nil, loc),
		"recap":       PeriodRecap("month", nil, nil, Gas{}, loc),
		"flow":        EnergyFlow(nil, nil, 0, 1, 0),
		"typicalDay":  TypicalDay(nil, nil, 0, 1, loc),
	}
	null := regexp.MustCompile(`"[A-Za-z]+":null`)
	for name, r := range reports {
		b, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range null.FindAllString(string(b), -1) {
			if m != `"tempFit":null` { // a fit that needs eight trips is a value that may be absent; every other null here would be a list
				t.Errorf("%s sends %s", name, m)
			}
		}
	}
}
